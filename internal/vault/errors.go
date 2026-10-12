package vault

import "errors"

// 领域错误类型。上层（GUI）可用 errors.Is 判断具体错误，给出用户友好的提示。
var (
	ErrSourceNotExist       = errors.New("源目录不存在")
	ErrSourceNotDir         = errors.New("源路径不是目录")
	ErrDecryptedDirNotExist = errors.New("解密目录不存在")
	ErrNothingToWriteBack   = errors.New("没有找到可写回的解密子目录")
	ErrReadDirMeta          = errors.New("读取目录元数据失败")
	ErrCreateDir            = errors.New("创建目录失败")
	ErrEncrypt              = errors.New("加密失败")
	ErrDecrypt              = errors.New("解密失败")
	ErrWriteBack            = errors.New("写回失败")
)
