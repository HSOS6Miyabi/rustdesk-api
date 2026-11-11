package service

import (
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// DirectoryUserKeyword narrows a previously authorized directory query.
func DirectoryUserKeyword(tx *gorm.DB, value string) *gorm.DB {
	keyword := strings.ToLower(strings.TrimSpace(value))
	if keyword == "" { return tx }
	id,_ := strconv.ParseUint(keyword,10,64)
	like := "%" + keyword + "%"
	return tx.Where("(id = ? OR LOWER(username) LIKE ? OR LOWER(nickname) LIKE ? OR LOWER(email) LIKE ?)", id,like,like,like)
}

// DirectoryPeerKeyword searches only within the directory scope supplied by the caller.
func DirectoryPeerKeyword(tx *gorm.DB, value string) *gorm.DB {
	keyword := strings.ToLower(strings.TrimSpace(value))
	if keyword == "" { return tx }
	userID,_ := strconv.ParseUint(keyword,10,64)
	like := "%" + keyword + "%"
	return tx.Where("(user_id = ? OR LOWER(id) LIKE ? OR LOWER(alias) LIKE ? OR LOWER(hostname) LIKE ? OR LOWER(username) LIKE ? OR LOWER(os) LIKE ? OR LOWER(last_online_ip) LIKE ?)",
		userID,like,like,like,like,like,like)
}
