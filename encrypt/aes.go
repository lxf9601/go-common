package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

type AesCryptor struct {
	Iv  []byte
	Key []byte
}

func (a *AesCryptor) Encrypt(data []byte) ([]byte, error) {
	aesBlockEncrypter, err := aes.NewCipher(a.Key)
	content := PKCS5Padding(data, aesBlockEncrypter.BlockSize())
	encrypted := make([]byte, len(content))
	if err != nil {
		println(err.Error())
		return nil, err
	}
	aesEncrypter := cipher.NewCBCEncrypter(aesBlockEncrypter, a.Iv)
	aesEncrypter.CryptBlocks(encrypted, content)
	return encrypted, nil
}

//解密数据
func (a *AesCryptor) Decrypt(src []byte) (data []byte, err error) {
	decrypted := make([]byte, len(src))
	var aesBlockDecrypter cipher.Block
	aesBlockDecrypter, err = aes.NewCipher(a.Key)
	if err != nil {
		println(err.Error())
		return nil, err
	}
	aesDecrypter := cipher.NewCBCDecrypter(aesBlockDecrypter, a.Iv)
 	aesDecrypter.CryptBlocks(decrypted, src)
	return PKCS5Trimming(decrypted), nil
}

// PKCS5Padding PKCS5包装
func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

// PKCS5Trimming 解包装
func PKCS5Trimming(encrypt []byte) []byte {
	if len(encrypt) > 0 {
		padding := encrypt[len(encrypt)-1]
		return encrypt[:len(encrypt)-int(padding)]
	} else {
		return []byte{}
	}
}
