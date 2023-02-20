package mashery

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"io"
	"math/big"
	"strconv"
	"time"
)

func retrievePrivateKey[T RoleContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {

	if found, pkBinary, err := reqCtx.ReadBinaryPath(ctx, rolePrivateKeyPath(reqCtx)); err != nil {
		return nil, err
	} else if found {
		reqCtx.heap.GetRole().PrivateKey = pkBinary
	} else if !found {
		if privateKey, err := rsa.GenerateKey(rand.Reader, 4096); err != nil {
			return nil, err
		} else {
			role := reqCtx.heap.GetRole()
			role.PrivateKey = x509.MarshalPKCS1PrivateKey(privateKey)
			if err = reqCtx.WriteBinaryPath(ctx, rolePrivateKeyPath(reqCtx), role.PrivateKey); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

func getPrivateKey(storedRole *StoredRole) (*rsa.PrivateKey, error) {
	if storedRole.PrivateKey == nil {
		return nil, errors.New("private key is not initialized for this role")
	}

	if key, err := x509.ParsePKCS1PrivateKey(storedRole.PrivateKey); err == nil {
		return key, nil
	}

	return nil, errors.New("private key data structure is not understood")
}

func renderRoleCertificate(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
	cn := "Bearer"
	if val, ok := reqCtx.data.GetOk(pemCommonNameField); ok {
		cn = val.(string)
	}

	template := createRoleCertificateTemplate(cn, time.Now(), time.Now().Add(time.Hour*4))

	return renderRoleCertificateWithTemplate(nil, reqCtx, reqCtx.heap.GetRole(), &template)
}

func createRoleCertificateTemplate(cn string, certFrom time.Time, certTo time.Time) x509.Certificate {
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"Mashery API HashiCorp Vault Authentication Backend"},
			CommonName:   cn,
		},
		NotBefore: certFrom, /*.Add(time.Minute)*/
		NotAfter:  certTo,

		KeyUsage:              x509.KeyUsageDataEncipherment,
		BasicConstraintsValid: true,
	}
	return template
}

func createSelfSignedCertificatePEMBlock(template *x509.Certificate, pk *rsa.PrivateKey, headerName string) (string, error) {
	if derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &pk.PublicKey, pk); err == nil {

		headers := map[string]string{
			"NotAfter":        template.NotAfter.String(),
			"Common-Name":     template.Subject.CommonName,
			roleNamePEMHeader: headerName,
		}

		blockText := createRecipientRolePEMBock(derBytes, headers)

		return blockText, nil
	} else {
		return "", errwrap.Wrapf("cannot generate x509 certificate ({{err}})", err)
	}
}

func createRecipientRolePEMBock(derBytes []byte, headers map[string]string) string {
	out := &bytes.Buffer{}
	_ = pem.Encode(out, &pem.Block{
		Type:    masheryRoleRecipientPEMBlockName,
		Bytes:   derBytes,
		Headers: headers,
	})
	blockText := out.String()
	return blockText
}

func renderRoleCertificateWithTemplate(_ context.Context, _ *RequestHandlerContext[RoleContext], role *StoredRole, template *x509.Certificate) (*logical.Response, error) {
	if role == nil {
		return nil, errors.New("role hasn't been initialized")
	}

	pk, err := getPrivateKey(role)
	if err != nil {
		return nil, err
	}

	if out, err := createSelfSignedCertificatePEMBlock(template, pk, role.Name); err != nil {
		return nil, errwrap.Wrapf("cannot generate x509 certificate ({{err}})", err)
	} else {
		resp := &logical.Response{
			Data: map[string]interface{}{
				pemContainerField: out,
			},
		}

		return resp, nil
	}
}

func readRecipientCertificate(_ context.Context, reqCtx *RequestHandlerContext[RoleExportContext]) (*logical.Response, error) {
	rawPEM := reqCtx.data.Get(pemContainerField).(string)
	if len(rawPEM) == 0 {
		return logical.ErrorResponse("no PEM-encoded data received"), nil
	}
	blk, _ := pem.Decode([]byte(rawPEM))
	if blk == nil {
		return logical.ErrorResponse("supplied PEM data bears no PEM block"), nil
	} else if blk.Type != masheryRoleRecipientPEMBlockName {
		return logical.ErrorResponse("input does not contain credentials recipient block"), nil
	}

	var recipientRole = "---not specified---"
	if len(blk.Headers[roleNamePEMHeader]) > 0 {
		recipientRole = blk.Headers[roleNamePEMHeader]
	}
	reqCtx.heap.CarryRecipientName(recipientRole)

	now := time.Now()
	if cert, err := x509.ParseCertificate(blk.Bytes); err != nil {
		return logical.ErrorResponse("received unparseable certificate: %s", err.Error()), nil
	} else if now.Before(cert.NotBefore) {
		return logical.ErrorResponse("supplied certificate is not yet valid"), nil
	} else if now.After(cert.NotAfter) {
		return logical.ErrorResponse("supplied certificate has already expired"), nil
	} else {
		reqCtx.heap.CarryRecipientCertificate(cert)
	}

	return nil, nil
}

type DesiredRoleExport struct {
	desiredTerm           time.Duration
	desiredNumUses        int
	desiredQps            int
	desiredForceProxyMode bool
	desiredOnlyV2         bool
	desiredOnlyV3         bool
	desireExportable      bool
}

func parseDesiredRoleExport(d *framework.FieldData) (DesiredRoleExport, error) {
	rv := DesiredRoleExport{
		0,
		-1,
		-1,
		false,
		false,
		false,
		false,
	}

	if v, ok := d.GetOk(explicitNumUsesField); ok {
		rv.desiredNumUses = v.(int)
	}
	if v, ok := d.GetOk(onlyV2Field); ok {
		rv.desiredOnlyV2 = v.(bool)
	}
	if v, ok := d.GetOk(onlyV3Field); ok {
		rv.desiredOnlyV3 = v.(bool)
	}
	if v, ok := d.GetOk(explicitQpsField); ok {
		rv.desiredQps = v.(int)
	}
	if v, ok := d.GetOk(forceProxyModeField); ok {
		rv.desiredForceProxyMode = v.(bool)
	}
	if v, ok := d.GetOk(exportableField); ok {
		rv.desireExportable = v.(bool)
	}

	if v, ok := d.GetOk(explicitTermField); ok {
		suppliedInput := v.(string)
		if dur, err := ParseUserInputDuration(suppliedInput); err != nil {
			return rv, err
		} else {
			rv.desiredTerm = dur
		}
	}

	return rv, nil
}

func GZipDecompress(input []byte) ([]byte, error) {
	reader := bytes.NewReader(input)
	gzreader, _ := gzip.NewReader(reader)
	defer gzreader.Close()

	return io.ReadAll(gzreader)
}

func GZipCompress(input []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	gz.Write(input)
	gz.Close()
	return b.Bytes()
}

func createRoleDataExchangePEMBlock(dat []byte, headers map[string]string) string {
	out := &bytes.Buffer{}

	_ = pem.Encode(out, &pem.Block{Type: masheryRoleDataPEMBlockName, Bytes: dat, Headers: headers})
	return out.String()
}

func renderEncryptedRoleData(_ context.Context, reqCtx *RequestHandlerContext[RoleExportContext]) (*logical.Response, error) {

	role := reqCtx.heap.GetRole()

	// Perform validation fo the parameters
	settings, err := parseDesiredRoleExport(reqCtx.data)
	if err != nil {
		return logical.ErrorResponse("invalid export configuration: %f", err), nil
	}

	exp := role.CreateRoleDataExchange(settings.desiredTerm)
	exp.RoleData.ForceProxyMode = settings.desiredForceProxyMode
	exp.RoleData.Exportable = settings.desireExportable

	if settings.desiredNumUses > 0 {
		exp.UsageTerm.ExplicitNumUses = int64(settings.desiredNumUses)

	}
	if settings.desiredQps > 0 {
		exp.RoleData.MaxQPS = settings.desiredQps
	}

	if settings.desiredOnlyV2 {
		exp.RoleData.AreaId = ""
		exp.RoleData.Username = ""
		exp.RoleData.Password = ""
	}
	if settings.desiredOnlyV3 {
		exp.RoleData.AreaNid = 0
	}

	jsonDat, _ := json.Marshal(&exp)

	dat, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, reqCtx.heap.GetRecipientCertificate().PublicKey.(*rsa.PublicKey), GZipCompress(jsonDat), reqCtx.plugin.cfg.OAEPLabel)
	if err != nil {
		return nil, err
	}

	var grantedNumUses = "∞"
	var grantedTerm = "∞"
	if exp.UsageTerm.ExplicitNumUses > 0 {
		grantedNumUses = fmt.Sprintf("max %d uses", exp.UsageTerm.ExplicitNumUses)
	}
	if exp.UsageTerm.ExplicitTerm > 0 {
		grantedTerm = time.Duration(exp.UsageTerm.ExplicitTerm).String()
	}

	pemOut := createRoleDataExchangePEMBlock(dat, map[string]string{
		"Date":              time.Now().String(),
		"Term":              grantedTerm,
		"Uses":              grantedNumUses,
		"Recipient":         reqCtx.heap.GetRecipientCertificate().Subject.String(),
		"Recipient Role":    reqCtx.heap.GetRecipientName(),
		"Origin Role":       role.Name,
		"V2 Capable":        strconv.FormatBool(exp.RoleData.IsV2Capable()),
		"V3 Capable":        strconv.FormatBool(exp.RoleData.IsV3Capable()),
		"Max QPS":           strconv.Itoa(exp.RoleData.MaxQPS),
		"Forced Proxy Mode": strconv.FormatBool(exp.RoleData.ForceProxyMode),
	})

	resp := &logical.Response{
		Data: map[string]interface{}{
			pemContainerField: pemOut,
		},
	}

	if settings.desiredTerm < 0 {
		resp.Warnings = []string{"explicit term is in the past"}
	}

	return resp, nil
}

func retrieveImportPEMBlockFromRequest(d *framework.FieldData) (*pem.Block, error) {
	pemStr := d.Get(pemContainerField).(string)
	if len(pemStr) == 0 {
		return nil, errors.New("empty PEM data received")
	}
	pemBlock, _ := pem.Decode([]byte(pemStr))
	if pemBlock == nil {
		return nil, errors.New(fmt.Sprintf("submitted data does not contain a valid PEM block"))
	}
	if pemBlock.Type != masheryRoleDataPEMBlockName {
		return nil, errors.New("incorrect PEM block")
	}

	return pemBlock, nil
}

func importPEMEncodedExchangeData(pemBlock *pem.Block) func(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
	return func(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
		pk, err := getPrivateKey(reqCtx.heap.GetRole())
		if err != nil {
			return nil, err
		}

		plainText, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, pemBlock.Bytes, reqCtx.plugin.cfg.OAEPLabel)
		if err != nil {
			return logical.ErrorResponse("was unable to decrypt the Mashery role data (%s)", err.Error()), nil
		}

		jsonTxt, err := GZipDecompress(plainText)
		if err != nil {
			return logical.ErrorResponse("was unable to GZipDecompress Mashery role data (%s)", err.Error()), nil
		}

		expRole := RoleDataExchange{}
		if err = json.Unmarshal(jsonTxt, &expRole); err != nil {
			return logical.ErrorResponse("was unable to parse Mashery role data (%s)", err.Error()), nil
		}

		reqCtx.heap.GetRole().Import(expRole)
		return nil, nil
	}
}
