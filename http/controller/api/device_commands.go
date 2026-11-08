package api

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

type DeviceCommands struct{}

type deviceAssignForm struct {
	Id                  string  `json:"id"`
	Uuid                string  `json:"uuid"`
	UserName            *string `json:"user_name"`
	StrategyName        *string `json:"strategy_name"`
	DeviceGroupName     *string `json:"device_group_name"`
	DeviceUsername      *string `json:"device_username"`
	DeviceName          *string `json:"device_name"`
	Note                *string `json:"note"`
	AddressBookName     *string `json:"address_book_name"`
	AddressBookTag      *string `json:"address_book_tag"`
	AddressBookAlias    *string `json:"address_book_alias"`
	AddressBookPassword *string `json:"address_book_password"`
	AddressBookNote     *string `json:"address_book_note"`
}

func nonEmptyDeviceID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 6 || len(value) > 253 {
		return false
	}
	return regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@:-]*$`).MatchString(value)
}

// Assign is compatible with the request body sent by RustDesk --assign.
func (d *DeviceCommands) Assign(c *gin.Context) {
	var f deviceAssignForm
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device assignment"})
		return
	}
	f.Id = strings.TrimSpace(f.Id)
	if !nonEmptyDeviceID(f.Id) || len(f.Uuid) == 0 || len(f.Uuid) > 256 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device ID or UUID is invalid"})
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user == nil || user.Id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	admin := service.AllService.UserService.IsAdmin(user)
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		var peer model.Peer
		if err := tx.Where("id = ? AND uuid = ?", f.Id, f.Uuid).First(&peer).Error; err != nil {
			return errors.New("registered device and UUID do not match")
		}
		if !admin && peer.UserId != user.Id {
			return errors.New("you may only update your own device")
		}
		updates := map[string]interface{}{}
		ownerId := peer.UserId
		if f.UserName != nil {
			targetName := strings.TrimSpace(*f.UserName)
			var target model.User
			if err := tx.Where("username = ? AND status = ?", targetName, model.COMMON_STATUS_ENABLE).First(&target).Error; err != nil {
				return errors.New("destination user was not found")
			}
			if !admin && target.Id != user.Id {
				return errors.New("assigning a device to another user requires administrator access")
			}
			ownerId = target.Id
			updates["user_id"] = target.Id
		}
		if f.DeviceGroupName != nil {
			name := strings.TrimSpace(*f.DeviceGroupName)
			if name == "" {
				updates["group_id"] = uint(0)
			} else {
				var group model.DeviceGroup
				if err := tx.Where("name = ?", name).First(&group).Error; err != nil {
					return errors.New("device group was not found")
				}
				updates["group_id"] = group.Id
			}
		}
		for _, field := range []struct {
			value *string
			name  string
			limit int
		}{
			{f.DeviceUsername, "username", 256},
			{f.DeviceName, "hostname", 256},
			{f.Note, "note", 2048},
		} {
			if field.value != nil {
				if len(*field.value) > field.limit {
					return fmt.Errorf("%s exceeds its length limit", field.name)
				}
				updates[field.name] = strings.TrimSpace(*field.value)
			}
		}
		if len(updates) != 0 {
			if err := tx.Model(&model.Peer{}).Where("row_id = ?", peer.RowId).Updates(updates).Error; err != nil {
				return err
			}
		}
		if f.StrategyName != nil {
			name := strings.TrimSpace(*f.StrategyName)
			if name == "" {
				if err := tx.Where("device_id = ?", f.Id).Delete(&model.DevicePolicyAssignment{}).Error; err != nil {
					return err
				}
			} else {
				var policy model.DevicePolicy
				if err := tx.Where("name = ?", name).First(&policy).Error; err != nil {
					return errors.New("device strategy was not found")
				}
				var assignment model.DevicePolicyAssignment
				err := tx.Where("device_id = ?", f.Id).First(&assignment).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					assignment = model.DevicePolicyAssignment{
						DeviceId: f.Id, PolicyId: policy.Id, Revision: time.Now().UnixNano(),
					}
					if err := tx.Create(&assignment).Error; err != nil {
						return err
					}
				} else if err != nil {
					return err
				} else if err := tx.Model(&assignment).Updates(map[string]interface{}{
					"policy_id": policy.Id, "revision": time.Now().UnixNano(),
				}).Error; err != nil {
					return err
				}
			}
		}
		if f.AddressBookName != nil {
			if ownerId == 0 {
				return errors.New("assign the device to a user before adding it to an address book")
			}
			name := strings.TrimSpace(*f.AddressBookName)
			if name == "" {
				return errors.New("address book name cannot be empty")
			}
			if len(name) > 100 {
				return errors.New("address book name is too long")
			}
			var collection model.AddressBookCollection
			err := tx.Where("user_id = ? AND name = ?", ownerId, name).First(&collection).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				collection = model.AddressBookCollection{UserId: ownerId, Name: name}
				if err := tx.Create(&collection).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			var entry model.AddressBook
			err = tx.Where("user_id = ? AND collection_id = ? AND id = ?", ownerId, collection.Id, f.Id).First(&entry).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				entry = model.AddressBook{UserId: ownerId, CollectionId: collection.Id, Id: f.Id, Tags: []byte("[]")}
				if err := tx.Create(&entry).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			entryUpdates := map[string]interface{}{}
			if f.AddressBookAlias != nil {
				entryUpdates["alias"] = strings.TrimSpace(*f.AddressBookAlias)
			}
			if f.AddressBookPassword != nil {
				entryUpdates["password"] = *f.AddressBookPassword
			}
			if f.AddressBookNote != nil {
				if len(*f.AddressBookNote) > 2048 {
					return errors.New("address book note is too long")
				}
				entryUpdates["note"] = strings.TrimSpace(*f.AddressBookNote)
			}
			if f.AddressBookTag != nil {
				tags := []string{}
				for _, raw := range strings.Split(*f.AddressBookTag, ",") {
					tag := strings.TrimSpace(raw)
					if tag != "" {
						tags = append(tags, tag)
					}
				}
				if len(tags) > 64 {
					return errors.New("too many address book tags")
				}
				encoded, err := json.Marshal(tags)
				if err != nil {
					return err
				}
				entryUpdates["tags"] = string(encoded)
			}
			if len(entryUpdates) > 0 {
				if err := tx.Model(&entry).Updates(entryUpdates).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

type deviceDeploymentRequest struct {
	Id   string `json:"id"`
	Uuid string `json:"uuid"`
	Pk   string `json:"pk"`
}

func normalizedBase64(value string, minSize, maxSize int) (string, bool) {
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		valueBytes, err := encoding.DecodeString(value)
		if err == nil && len(valueBytes) >= minSize && len(valueBytes) <= maxSize {
			return base64.StdEncoding.EncodeToString(valueBytes), true
		}
	}
	return "", false
}

func validateDeployment(f *deviceDeploymentRequest) bool {
	f.Id = strings.TrimSpace(f.Id)
	if !nonEmptyDeviceID(f.Id) {
		return false
	}
	var ok bool
	f.Uuid, ok = normalizedBase64(f.Uuid, 8, 128)
	if !ok {
		return false
	}
	f.Pk, ok = normalizedBase64(f.Pk, 32, 32)
	return ok
}

// Deploy responds with the result strings expected by the official RustDesk client.
func (d *DeviceCommands) Deploy(c *gin.Context) {
	if !global.Config.App.RequireDeviceDeployment {
		c.JSON(http.StatusOK, gin.H{"result": "NOT_ENABLED"})
		return
	}
	if global.Config.App.InternalKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"result": "SERVER_ERROR"})
		return
	}
	var f deviceDeploymentRequest
	if err := c.ShouldBindJSON(&f); err != nil || !validateDeployment(&f) {
		c.JSON(http.StatusOK, gin.H{"result": "INVALID_INPUT"})
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user == nil || user.Id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"result": "UNAUTHORIZED"})
		return
	}
	result := "OK"
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		var peer model.Peer
		err := tx.Where("id = ?", f.Id).First(&peer).Error
		if err == nil && peer.Uuid != "" {
			normalized, valid := normalizedBase64(peer.Uuid, 8, 128)
			if !valid || normalized != f.Uuid {
				result = "ID_TAKEN"
				return nil
			}
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var existing model.DeviceDeployment
		err = tx.Where("device_id = ?", f.Id).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&model.DeviceDeployment{
				DeviceId: f.Id, Uuid: f.Uuid, PublicKey: f.Pk, OwnerUserId: user.Id, Status: "approved",
			}).Error
		}
		if err != nil {
			return err
		}
		if existing.Uuid != f.Uuid || existing.PublicKey != f.Pk {
			result = "ID_TAKEN"
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"result": "SERVER_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

// AuthorizeRegistration is restricted to hbbs via a shared internal key.
func (d *DeviceCommands) AuthorizeRegistration(c *gin.Context) {
	expected := global.Config.App.InternalKey
	provided := c.GetHeader("X-RustDesk-Internal-Key")
	if expected == "" || provided == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		c.JSON(http.StatusForbidden, gin.H{"result": "DENIED"})
		return
	}
	if !global.Config.App.RequireDeviceDeployment {
		c.JSON(http.StatusOK, gin.H{"result": "NOT_ENABLED"})
		return
	}
	var f deviceDeploymentRequest
	if err := c.ShouldBindJSON(&f); err != nil || !validateDeployment(&f) {
		c.JSON(http.StatusBadRequest, gin.H{"result": "INVALID_INPUT"})
		return
	}
	var deployment model.DeviceDeployment
	err := service.DB.Where("device_id = ?", f.Id).First(&deployment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, gin.H{"result": "NOT_DEPLOYED"})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"result": "SERVER_ERROR"})
		return
	}
	if deployment.Status != "approved" || deployment.Uuid != f.Uuid || deployment.PublicKey != f.Pk {
		c.JSON(http.StatusOK, gin.H{"result": "ID_TAKEN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}
