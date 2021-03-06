package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

const (
	// ModePerm is 777 for created dir shared with other docker
	ModePerm        os.FileMode = 0777
	mountDirPathKey string      = "MOUNT_PATH"
)

// GetFunctionSettingPath will return <appid>.property path of appid
func GetFunctionSettingPath(appid string) string {
	configPath := fmt.Sprintf("%s/%s.property", getAppidDir(appid), appid)
	return configPath
}

func GetFunctionSettingTmpPath(appid string) string {
	configPath := fmt.Sprintf("%s/%s_function_tmp.property", getAppidDir(appid), appid)
	return configPath
}

func SaveDictionaryFile(appid string, filename string, file multipart.File) (int64, error) {
	dirPath := getAppidDir(appid)
	filePath := fmt.Sprintf("%s/%s", dirPath, filename)

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		mkdirErr := os.MkdirAll(dirPath, ModePerm)
		if mkdirErr != nil {
			LogError.Printf("Cannot create appid dir into system (%s)", mkdirErr.Error())
			return 0, mkdirErr
		}
	}

	output, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		LogError.Printf("Cannot create file (%s)", err.Error())
		return 0, err
	}
	defer output.Close()

	written, err := io.Copy(output, file)
	if err != nil {
		LogError.Printf("Cannot copy file into system (%s)", err.Error())
		return 0, err
	}

	LogTrace.Printf("Write to file [%s] [%d] bytes successfully", filePath, written)
	return written, nil
}

func getAppidDir(appid string) string {
	mountPath := getGlobalEnv(mountDirPathKey)
	LogTrace.Printf("Mount path: %s", mountPath)

	dirPath := fmt.Sprintf("%s/settings/%s", mountPath, appid)
	mkdirErr := os.MkdirAll(dirPath, ModePerm)
	if mkdirErr != nil {
		LogError.Printf("Cannot create appid dir into system (%s)", mkdirErr.Error())
	}
	return dirPath
}

func GetMountDir() string {
	mountPath := getGlobalEnv(mountDirPathKey)
	LogTrace.Printf("Mount path: %s", mountPath)
	return mountPath
}
