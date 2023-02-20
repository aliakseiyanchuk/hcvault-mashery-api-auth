package mashery

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestGZipCompression(t *testing.T) {
	data := "abcdefg"
	gzippedData := GZipCompress([]byte(data))
	rehydratedData, err := GZipDecompress(gzippedData)

	assert.Nil(t, err)
	assert.Equal(t, data, string(rehydratedData))
}

func TestRetrievePrivateKey_FailsOnReadError(t *testing.T) {
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Get", mock.Anything, rolePrivateKeyPath(reqCtx)).
		Return(nil, errors.New("sample rejection"))

	lr, err := retrievePrivateKey(context.TODO(), reqCtx)
	emulStore.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to read data from storage: sample rejection", err.Error())
}

func TestRetrievePrivateKey_ReturnsOnSuccessfulRead(t *testing.T) {
	pkData := randomPrivateKey()
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Get", mock.Anything, rolePrivateKeyPath(reqCtx)).
		Return(createBinaryStorageEntryFrom(t, rolePrivateKeyPath(reqCtx), pkData), nil)

	lr, err := retrievePrivateKey(context.TODO(), reqCtx)
	emulStore.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.Nil(t, err)
	assert.Equal(t, pkData, reqCtx.heap.GetRole().PrivateKey)
}

func TestRetrievePrivateKey_GeneratesKeyIfMissing(t *testing.T) {
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Get", mock.Anything, rolePrivateKeyPath(reqCtx)).
		Return(nil, nil)
	emulStore.
		On("Put", mock.Anything, storageEntryBearingData()).
		Return(nil)

	lr, err := retrievePrivateKey(context.TODO(), reqCtx)
	emulStore.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.Nil(t, err)
	assert.True(t, len(reqCtx.heap.GetRole().PrivateKey) > 0)
}

func TestRetrievePrivateKey_WillErrIfPrivateKeyWriteWillFail(t *testing.T) {
	sr := StoredRole{
		Usage: StoredRoleUsage{
			RemainingNumUses: 345,
		},
	}

	emulStore, reqCtx := setupRoleRequestMockHaving(sr)
	emulStore.
		On("Get", mock.Anything, rolePrivateKeyPath(reqCtx)).
		Return(nil, nil)
	emulStore.
		On("Put", mock.Anything, storageEntryBearingData()).
		Return(errors.New("sample rejection"))

	lr, err := retrievePrivateKey(context.TODO(), reqCtx)
	emulStore.AssertExpectations(t)

	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "sample rejection", err.Error())
}

func TestGetPrivateKey_WhenNotInitialized(t *testing.T) {
	role := StoredRole{}

	key, err := getPrivateKey(&role)
	assert.Nil(t, key)
	assert.NotNil(t, err)
	assert.Equal(t, "private key is not initialized for this role", err.Error())
}

func TestGetPrivateKey_WhenMalformed(t *testing.T) {
	role := StoredRole{
		PrivateKey: []byte("mailformed"),
	}

	key, err := getPrivateKey(&role)
	assert.Nil(t, key)
	assert.NotNil(t, err)
	assert.Equal(t, "private key data structure is not understood", err.Error())
}

func TestGetPrivateKey_WhenValid(t *testing.T) {
	pkData := randomPrivateKey()

	role := StoredRole{
		PrivateKey: pkData,
	}

	key, err := getPrivateKey(&role)
	assert.NotNil(t, key)
	assert.Nil(t, err)
}

func randomPrivateKey() []byte {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 4096)
	pkData := x509.MarshalPKCS1PrivateKey(privateKey)
	return pkData
}

func TestRenderRoleCertificate_WhenPrimaryKeyIsBad(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &StoredRole{},
		},
		fieldSchema: pathRolePemReadFields,
	}
	_, reqCtx := mockBuilder.Build()

	lr, err := renderRoleCertificate(nil, reqCtx)
	assert.Nil(t, lr)
	assert.NotNil(t, err)
}

func TestRenderRoleCertificate_Default(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &StoredRole{
				Name:       "sample",
				PrivateKey: randomPrivateKey(),
			},
		},
		fieldSchema: pathRolePemReadFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := renderRoleCertificate(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.True(t, len(lr.Data[pemContainerField].(string)) > 0)
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_WithNoInput(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container:   &RoleExportContainer{},
		fieldSchema: pathRoleExportFields,
	}
	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "no PEM-encoded data received", lr.Error().Error())
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_WithInvalidBlock(t *testing.T) {
	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: "malformed input",
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "supplied PEM data bears no PEM block", lr.Error().Error())
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_UnexpectedBlockType(t *testing.T) {
	outBlock := &bytes.Buffer{}
	_ = pem.Encode(outBlock, &pem.Block{
		Type:  "invalid block type",
		Bytes: []byte("unsupported-context"),
	})

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: outBlock.String(),
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.NotNil(t, lr)
	assert.Equal(t, "input does not contain credentials recipient block", lr.Error().Error())
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_HappyPath(t *testing.T) {
	// Extract the source certificate
	_, pemData := testAutoGeneratedSourceRoleCert(t, time.Now(), time.Now().Add(time.Second*20))

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: pemData,
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_BlockCertInFuture(t *testing.T) {
	// Extract the source certificate
	_, pemData := testAutoGeneratedSourceRoleCert(t, time.Now().Add(time.Minute*10), time.Now().Add(time.Minute*20))

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: pemData,
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.Equal(t, "supplied certificate is not yet valid", lr.Error().Error())
	assert.NotNil(t, lr)
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_BlockExpiredCerts(t *testing.T) {
	// Extract the source certificate
	_, pemData := testAutoGeneratedSourceRoleCert(t, time.Now().Add(-1*time.Minute*20), time.Now().Add(-1*time.Minute*10))

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: pemData,
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.Equal(t, "supplied certificate has already expired", lr.Error().Error())
	assert.NotNil(t, lr)
	assert.Nil(t, err)
}

func TestReadRecipientCertificate_GarbagePEMBock(t *testing.T) {
	// Extract the source certificate
	pemData := createRecipientRolePEMBock([]byte("garbage string"), map[string]string{})

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{},
		data: map[string]interface{}{
			pemContainerField: pemData,
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.Equal(t, "received unparseable certificate: x509: malformed certificate", lr.Error().Error())
	assert.NotNil(t, lr)
	assert.Nil(t, err)
}

func testAutoGeneratedSourceRoleCert(t *testing.T, from time.Time, to time.Time) (*rsa.PrivateKey, interface{}) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 4096)

	template := createRoleCertificateTemplate("test", from, to)
	pemData, err := createSelfSignedCertificatePEMBlock(&template, privateKey, "test role")
	assert.Nil(t, err)

	return privateKey, pemData
}

func TestParseDesiredRoleExport_WithEmptyConfig(t *testing.T) {
	_, fullData := setupRoleRequestMockWithData(map[string]interface{}{}, pathRoleExportFields)

	cfg, err := parseDesiredRoleExport(fullData.data)
	assert.Nil(t, err)

	assert.Equal(t, -1, cfg.desiredNumUses)
	assert.False(t, cfg.desiredOnlyV2)
	assert.False(t, cfg.desiredOnlyV3)
	assert.Equal(t, -1, cfg.desiredQps)
	assert.False(t, cfg.desiredForceProxyMode)
	assert.False(t, cfg.desireExportable)
	assert.Equal(t, time.Duration(0), cfg.desiredTerm)
}

func TestParseDesiredRoleExport_WithFullConfig(t *testing.T) {
	_, fullData := setupRoleRequestMockWithData(map[string]interface{}{
		explicitNumUsesField: 23,
		onlyV2Field:          false,
		onlyV3Field:          false,
		explicitQpsField:     45,
		forceProxyModeField:  true,
		exportableField:      true,
		explicitTermField:    "2w",
	}, pathRoleExportFields)

	cfg, err := parseDesiredRoleExport(fullData.data)
	assert.Nil(t, err)

	assert.Equal(t, 23, cfg.desiredNumUses)
	assert.False(t, cfg.desiredOnlyV2)
	assert.False(t, cfg.desiredOnlyV3)
	assert.Equal(t, 45, cfg.desiredQps)
	assert.True(t, cfg.desiredForceProxyMode)
	assert.True(t, cfg.desireExportable)
	assert.True(t, cfg.desiredTerm > 0)
}

func TestParseDesiredRoleExport_GarbageTime(t *testing.T) {
	_, fullData := setupRoleRequestMockWithData(map[string]interface{}{
		explicitTermField: "garbage time",
	}, pathRoleExportFields)

	_, err := parseDesiredRoleExport(fullData.data)
	assert.NotNil(t, err)
	assert.Equal(t, "time: invalid duration \"garbage time\"", err.Error())
}

func TestRenderEncryptedRoleData_GarbageTime(t *testing.T) {
	_, fullData := setupRoleRequestMockWithData(map[string]interface{}{
		explicitTermField: "garbage time",
	}, pathRoleExportFields)

	_, err := parseDesiredRoleExport(fullData.data)
	assert.NotNil(t, err)
	assert.Equal(t, "time: invalid duration \"garbage time\"", err.Error())
}

func TestRenderEncryptedRoleData_HappyFlow(t *testing.T) {
	_, pemBlock := testAutoGeneratedSourceRoleCert(t, time.Now(), time.Now().Add(time.Minute*5))
	role := createRoleWithFilledRoleKeys()

	mockBuilder := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{
			RoleContainer: RoleContainer{
				role: &role,
			},
		},
		data: map[string]interface{}{
			pemContainerField: pemBlock,
		},
		fieldSchema: pathRoleExportFields,
	}

	_, reqCtx := mockBuilder.Build()

	// The certificate for the export must be read
	lr, err := readRecipientCertificate(nil, reqCtx)
	assert.Nil(t, lr)
	assert.Nil(t, err)

	lr, err = renderEncryptedRoleData(nil, reqCtx)

	assert.NotNil(t, lr)
	assert.True(t, len(lr.Data[pemContainerField].(string)) > 0)
	assert.Nil(t, err)
}

func createRoleWithFilledRoleKeys() StoredRole {
	return StoredRole{
		Keys: RoleKeys{
			AreaNid:    456,
			AreaId:     "a-b-c-d",
			ApiKey:     "key",
			KeySecret:  "secret",
			Username:   "user",
			Password:   "pwd",
			MaxQPS:     34,
			Exportable: true,
			Imported:   false,
		},
	}
}

func TestRetrieveImportPEMBlockFromRequest_RejectsEmptyString(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{}, pathRolePemImportFields)
	block, err := retrieveImportPEMBlockFromRequest(reqCtx.data)
	assert.Nil(t, block)
	assert.NotNil(t, err)
	assert.Equal(t, "empty PEM data received", err.Error())
}

func TestRetrieveImportPEMBlockFromRequest_RejectsMalformedString(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		pemContainerField: "malformed-string",
	}, pathRolePemImportFields)
	block, err := retrieveImportPEMBlockFromRequest(reqCtx.data)
	assert.Nil(t, block)
	assert.NotNil(t, err)
	assert.Equal(t, "submitted data does not contain a valid PEM block", err.Error())
}

func TestRetrieveImportPEMBlockFromRequest_RejectsBlockOfUnexpectedType(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		pemContainerField: createRecipientRolePEMBock([]byte("garbage"), map[string]string{}),
	}, pathRolePemImportFields)
	// The recipient role block is not valid

	block, err := retrieveImportPEMBlockFromRequest(reqCtx.data)
	assert.Nil(t, block)
	assert.NotNil(t, err)
	assert.Equal(t, "incorrect PEM block", err.Error())
}

func TestRetrieveImportPEMBlockFromRequest_HappyPatch(t *testing.T) {
	_, reqCtx := setupRoleRequestMockWithData(map[string]interface{}{
		pemContainerField: createRoleDataExchangePEMBlock([]byte("garbage"), map[string]string{}),
	}, pathRolePemImportFields)
	// The recipient role block is not valid

	block, err := retrieveImportPEMBlockFromRequest(reqCtx.data)
	assert.NotNil(t, block)
	assert.Nil(t, err)
}

func TestImportPEMBlock_WillFailOnMissingPrivateKey(t *testing.T) {
	_, reqCtx := setupRoleRequestMockHaving(StoredRole{})
	lr, err := importPEMEncodedExchangeData(nil)(nil, reqCtx)
	assert.Nil(t, lr)
	assert.NotNil(t, err)
	assert.Equal(t, "private key is not initialized for this role", err.Error())
}

func setupTestRoleDataExport() (StoredRole, StoredRole, *pem.Block) {
	sourceReq := RoleRequestMockBuilder[RoleContext]{
		container: &RoleContainer{
			role: &StoredRole{
				PrivateKey: randomPrivateKey(),
			},
		},
		fieldSchema: pathRoleExportFields,
	}

	// Simulate exporting certificate from source role
	_, rolePEMReadRequest := sourceReq.Build()
	lr, _ := renderRoleCertificate(nil, rolePEMReadRequest)

	exportRole := createRoleWithFilledRoleKeys()
	exportReq := RoleRequestMockBuilder[RoleExportContext]{
		container: &RoleExportContainer{
			RoleContainer: RoleContainer{
				role: &exportRole,
			},
		},
		fieldSchema: pathRoleExportFields,
		data: map[string]interface{}{
			pemContainerField: lr.Data[pemContainerField],
		},
	}

	_, exportRoleRequest := exportReq.Build()

	// Simulate exporting role data
	readRecipientCertificate(nil, exportRoleRequest)
	lr, _ = renderEncryptedRoleData(nil, exportRoleRequest)

	pemOut, _ := pem.Decode([]byte(lr.Data[pemContainerField].(string)))

	return *rolePEMReadRequest.heap.GetRole(), *exportRoleRequest.heap.GetRole(), pemOut
}

func TestImportPEMBlock_WillFailOnMismatchingEncryption(t *testing.T) {
	_, _, pemOut := setupTestRoleDataExport()

	confusedSourceRole := StoredRole{
		PrivateKey: randomPrivateKey(),
	}

	_, importRoleRequest := setupRoleRequestMockHaving(confusedSourceRole)

	lr, err := importPEMEncodedExchangeData(pemOut)(nil, importRoleRequest)

	assert.NotNil(t, lr)
	assert.Equal(t, "was unable to decrypt the Mashery role data (crypto/rsa: decryption error)", lr.Error().Error())

	assert.Nil(t, err)
}

func TestImportPEMBlock_WillImport(t *testing.T) {
	sourceRole, exportedRole, pemOut := setupTestRoleDataExport()

	// Now, let's try to decrypt it with the confused role
	_, importRoleRequest := setupRoleRequestMockHaving(sourceRole)
	lr, err := importPEMEncodedExchangeData(pemOut)(nil, importRoleRequest)

	assert.Nil(t, lr)
	assert.Nil(t, err)

	exportedKeys := exportedRole.Keys
	importedKeys := importRoleRequest.heap.GetRole().Keys

	assert.Equal(t, exportedKeys.AreaNid, importedKeys.AreaNid)
	assert.Equal(t, exportedKeys.AreaId, importedKeys.AreaId)
	assert.Equal(t, exportedKeys.ApiKey, importedKeys.ApiKey)
	assert.Equal(t, exportedKeys.KeySecret, importedKeys.KeySecret)
	assert.Equal(t, exportedKeys.Username, importedKeys.Username)
	assert.Equal(t, exportedKeys.Password, importedKeys.Password)
	assert.Equal(t, exportedKeys.MaxQPS, importedKeys.MaxQPS)

	assert.True(t, importedKeys.Imported)
	assert.False(t, importedKeys.Exportable)

	assert.True(t, importRoleRequest.heap.GetRole().Usage.IsUnboundedUsage())
}
