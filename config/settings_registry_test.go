package config

import (
    "strings"
    "testing"
)

func TestAllConfigFieldsHaveReadableDescriptions(t *testing.T) {
    settings:=SettingDefinitions()
    if len(settings)<90 {t.Fatalf("configuration registry is too small: %d",len(settings))}
    for _,item:=range settings {
        if item.Name==""||item.Name==item.Key||
            strings.HasPrefix(item.Description,"应用配置项 ")||item.Description==""{
            t.Errorf("configuration description missing: %s",item.Key)
        }
    }
}
