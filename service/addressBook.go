package service

import (
	"encoding/json"
	"errors"
	"reflect"
	"github.com/google/uuid"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"strings"
)

type AddressBookService struct {
}

func (s *AddressBookService) Info(id string) *model.AddressBook {
	p := &model.AddressBook{}
	DB.Where("id = ?", id).First(p)
	return p
}

func (s *AddressBookService) InfoByUserIdAndId(userid uint, id string) *model.AddressBook {
	p := &model.AddressBook{}
	DB.Where("user_id = ? and id = ?", userid, id).First(p)
	return p
}

func (s *AddressBookService) InfoByUserIdAndIdAndCid(userid uint, id string, cid uint) *model.AddressBook {
	p := &model.AddressBook{}
	DB.Where("user_id = ? and id = ? and collection_id = ?", userid, id, cid).First(p)
	return p
}
func (s *AddressBookService) InfoByRowId(id uint) *model.AddressBook {
	p := &model.AddressBook{}
	DB.Where("row_id = ?", id).First(p)
	return p
}
func (s *AddressBookService) ListByUserId(userId, page, pageSize uint) (res *model.AddressBookList) {
	res = s.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ?", userId)
	})
	return
}
func (s *AddressBookService) ListByUserIds(userIds []uint, page, pageSize uint) (res *model.AddressBookList) {
	res = s.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("user_id in (?)", userIds)
	})
	return
}

// AddAddressBook
func (s *AddressBookService) AddAddressBook(ab *model.AddressBook) error {
	return DB.Create(ab).Error
}

// sameAddressBookContent ignores database metadata and compares the client
// fields that are actually persisted by an address-book synchronization.
func sameAddressBookContent(previous, current *model.AddressBook) bool {
	if previous==nil||current==nil {return false}
	a,b:=*previous,*current
	b.RowId=a.RowId
	b.TimeModel=a.TimeModel
	b.DeletedAt=a.DeletedAt
	b.Collection=a.Collection
	return reflect.DeepEqual(a,b)
}

// UpdateAddressBook synchronizes only changed entries, reading peer metadata in
// bounded batches and preserving the existing atomic replacement behavior.
func (s *AddressBookService) UpdateAddressBook(abs []*model.AddressBook, userId uint) error {
	if userId==0 {return errors.New("user ID is required")}
	if len(abs)>5000 {return errors.New("too many address book entries")}
	seen:=make(map[string]bool,len(abs))
	needsPeer:=make([]string,0)
	for _,item:=range abs {
		if item==nil||strings.TrimSpace(item.Id)=="" {
			return errors.New("device ID cannot be empty")
		}
		if seen[item.Id] {return errors.New("duplicate device ID")}
		seen[item.Id]=true
		if item.Platform==""||item.Username==""||item.Hostname==""{
			needsPeer=append(needsPeer,item.Id)
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var saved []*model.AddressBook
		if err:=tx.Where("user_id = ? AND collection_id = 0",userId).
			Find(&saved).Error;err!=nil{return err}
		existing:=make(map[string]*model.AddressBook,len(saved))
		for _,entry:=range saved{existing[entry.Id]=entry}
		peerByID:=make(map[string]model.Peer,len(needsPeer))
		for start:=0;start<len(needsPeer);start+=250{
			end:=start+250
			if end>len(needsPeer){end=len(needsPeer)}
			var peers []model.Peer
			if err:=tx.Model(&model.Peer{}).
				Select("id,os,username,hostname,last_online_time,row_id").
				Where("id IN ?",needsPeer[start:end]).
				Order("last_online_time DESC, row_id DESC").
				Find(&peers).Error;err!=nil{return err}
			for _,peer:=range peers {
				if _,ok:=peerByID[peer.Id];!ok {peerByID[peer.Id]=peer}
			}
		}
		added:=make([]*model.AddressBook,0)
		for _,item:=range abs {
			item.UserId=userId
			item.CollectionId=0
			if peer,ok:=peerByID[item.Id];ok {
				if item.Platform==""{item.Platform=s.PlatformFromOs(peer.Os)}
				if item.Username==""{item.Username=peer.Username}
				if item.Hostname==""{item.Hostname=peer.Hostname}
			}
			if previous,ok:=existing[item.Id];ok {
				if !sameAddressBookContent(previous,item) {
					if err:=tx.Model(&model.AddressBook{}).
						Where("row_id = ?",previous.RowId).
						Select("*").Omit("row_id","created_at","updated_at").
						Updates(item).Error;err!=nil{return err}
				}
				delete(existing,item.Id)
			}else{
				added=append(added,item)
			}
		}
		if len(added)>0 {
			if err:=tx.CreateInBatches(added,100).Error;err!=nil{return err}
		}
		toDelete:=make([]uint,0,len(existing))
		for _,entry:=range existing{toDelete=append(toDelete,entry.RowId)}
		for start:=0;start<len(toDelete);start+=250{
			end:=start+250
			if end>len(toDelete){end=len(toDelete)}
			if err:=tx.Where("row_id IN ?",toDelete[start:end]).
				Delete(&model.AddressBook{}).Error;err!=nil{return err}
		}
		return nil
	})
}

func (s *AddressBookService) List(page, pageSize uint, where func(tx *gorm.DB)) (res *model.AddressBookList) {
	res = &model.AddressBookList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.AddressBook{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.AddressBooks)
	return
}

func (s *AddressBookService) FromPeer(peer *model.Peer) (a *model.AddressBook) {
	a = &model.AddressBook{}
	a.Id = peer.Id
	a.Username = peer.Username
	a.Hostname = peer.Hostname
	a.UserId = peer.UserId
	a.Platform = s.PlatformFromOs(peer.Os)
	return a
}

// Create 创建
func (s *AddressBookService) Create(u *model.AddressBook) error {
	res := DB.Create(u).Error
	return res
}
func (s *AddressBookService) Delete(u *model.AddressBook) error {
	return DB.Delete(u).Error
}

// Update 更新
func (s *AddressBookService) Update(u *model.AddressBook) error {
	return DB.Model(u).Updates(u).Error
}

// UpdateByMap 更新
func (s *AddressBookService) UpdateByMap(u *model.AddressBook, data map[string]interface{}) error {
	return DB.Model(u).Updates(data).Error
}

// UpdateAll 更新
func (s *AddressBookService) UpdateAll(u *model.AddressBook) error {
	return DB.Model(u).Select("*").Omit("created_at").Updates(u).Error
}

// ShareByWebClient 分享
func (s *AddressBookService) ShareByWebClient(m *model.ShareRecord) error {
	m.ShareToken = uuid.New().String()
	return DB.Create(m).Error
}

// SharedPeer
func (s *AddressBookService) SharedPeer(shareToken string) *model.ShareRecord {
	m := &model.ShareRecord{}
	DB.Where("share_token = ?", shareToken).First(m)
	return m
}

// PlatformFromOs
func (s *AddressBookService) PlatformFromOs(os string) string {
	if strings.Contains(os, "Android") || strings.Contains(os, "android") {
		return "Android"
	}
	if strings.Contains(os, "Windows") || strings.Contains(os, "windows") {
		return "Windows"
	}
	if strings.Contains(os, "Linux") || strings.Contains(os, "linux") {
		return "Linux"
	}
	if strings.Contains(os, "mac") || strings.Contains(os, "Mac") {
		return "Mac OS"
	}
	return ""
}
func (s *AddressBookService) ListByUserIdAndCollectionId(userId, cid, page, pageSize uint) (res *model.AddressBookList) {
	res = s.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ? and collection_id = ?", userId, cid)
	})
	return
}
func (s *AddressBookService) ListCollection(page, pageSize uint, where func(tx *gorm.DB)) (res *model.AddressBookCollectionList) {
	res = &model.AddressBookCollectionList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.AddressBookCollection{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.AddressBookCollection)
	return
}
func (s *AddressBookService) ListCollectionByIds(ids []uint) (res []*model.AddressBookCollection) {
	DB.Where("id in ?", ids).Find(&res)
	return res
}

func (s *AddressBookService) ListCollectionByUserId(userId uint) (res *model.AddressBookCollectionList) {
	res = s.ListCollection(1, 100, func(tx *gorm.DB) {
		tx.Where("user_id = ?", userId)
	})
	return
}
func (s *AddressBookService) CollectionInfoById(id uint) *model.AddressBookCollection {
	p := &model.AddressBookCollection{}
	DB.Where("id = ?", id).First(p)
	return p
}

// CollectionReadRules locates shared address books for an account using its
// primary group, additional memberships, and public sharing rules.
func (s *AddressBookService) CollectionReadRules(user *model.User) (res []*model.AddressBookCollectionRule) {
	if user == nil || user.Id == 0 {
		return nil
	}
	if err := DB.Where("type = ? AND to_id = ? AND rule > 0",
		model.ShareAddressBookRuleTypePersonal, user.Id).Find(&res).Error; err != nil {
		return nil
	}
	groups, err := (&GroupService{}).GroupIdsForUser(user)
	if err != nil {
		return nil
	}
	if len(groups) > 0 {
		var groupRules []*model.AddressBookCollectionRule
		if err := DB.Where("type = ? AND to_id IN ? AND rule > 0",
			model.ShareAddressBookRuleTypeGroup, groups).Find(&groupRules).Error; err != nil {
			return nil
		}
		res = append(res, groupRules...)
	}
	var publicRules []*model.AddressBookCollectionRule
	if err := DB.Where("type = ? AND to_id = 0 AND rule > 0",
		model.ShareAddressBookRuleTypeAll).Find(&publicRules).Error; err != nil {
		return nil
	}
	return append(res, publicRules...)
}

// BatchUserMaxRules determines shared collection permissions from one rule
// query and one membership query, while keeping the existing precedence.
func (s *AddressBookService) BatchUserMaxRules(user *model.User, collections []*model.AddressBookCollection) (map[uint]int, error) {
	values := make(map[uint]int, len(collections))
	if user == nil || user.Id == 0 || len(collections) == 0 {
		return values, nil
	}
	ids := make([]uint, 0, len(collections))
	byID := make(map[uint]*model.AddressBookCollection, len(collections))
	for _, collection := range collections {
		if collection == nil || collection.Id == 0 { continue }
		ids = append(ids, collection.Id)
		byID[collection.Id] = collection
		if collection.UserId == user.Id {
			values[collection.Id] = model.ShareAddressBookRuleRuleFullControl
		}
	}
	if len(ids) == 0 { return values, nil }
	groupIDs, err := (&GroupService{}).GroupIdsForUser(user)
	if err != nil { return nil, err }
	groupSet := make(map[uint]bool, len(groupIDs))
	for _, id := range groupIDs { groupSet[id] = true }
	var rules []model.AddressBookCollectionRule
	if err := DB.Where("collection_id IN ?", ids).Find(&rules).Error; err != nil {
		return nil, err
	}
	type ruleChoice struct {
		personalID uint
		personalRule int
		groupRule int
		publicID uint
		publicRule int
	}
	choices := make(map[uint]ruleChoice, len(ids))
	for _, rule := range rules {
		collection := byID[rule.CollectionId]
		if collection == nil || collection.UserId != rule.UserId || collection.UserId == user.Id {
			continue
		}
		choice := choices[rule.CollectionId]
		switch {
		case rule.Type == model.ShareAddressBookRuleTypePersonal && rule.ToId == user.Id:
			if rule.Id >= choice.personalID {
				choice.personalID, choice.personalRule = rule.Id, rule.Rule
			}
		case rule.Type == model.ShareAddressBookRuleTypeGroup && groupSet[rule.ToId]:
			if rule.Rule > choice.groupRule { choice.groupRule = rule.Rule }
		case rule.Type == model.ShareAddressBookRuleTypeAll && rule.ToId == 0:
			if rule.Id >= choice.publicID {
				choice.publicID, choice.publicRule = rule.Id, rule.Rule
			}
		}
		choices[rule.CollectionId] = choice
	}
	for _, collection := range collections {
		if collection == nil || collection.UserId == user.Id { continue }
		choice := choices[collection.Id]
		switch {
		case choice.personalID > 0:
			values[collection.Id] = choice.personalRule
		case choice.groupRule > 0:
			values[collection.Id] = choice.groupRule
		default:
			values[collection.Id] = choice.publicRule
		}
	}
	return values, nil
}

// UserMaxRule chooses the most specific applicable sharing rule. Multiple
// memberships in different groups combine by choosing the strongest group rule.
func (s *AddressBookService) UserMaxRule(user *model.User, ownerID, collectionID uint) int {
	if user == nil || user.Id == 0 || ownerID == 0 {
		return 0
	}
	if user.Id == ownerID {
		return model.ShareAddressBookRuleRuleFullControl
	}
	var direct model.AddressBookCollectionRule
	err := DB.Where("user_id = ? AND collection_id = ? AND type = ? AND to_id = ?",
		ownerID, collectionID, model.ShareAddressBookRuleTypePersonal, user.Id).
		Order("id DESC").First(&direct).Error
	if err == nil {
		return direct.Rule
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0
	}
	groups, err := (&GroupService{}).GroupIdsForUser(user)
	if err != nil {
		return 0
	}
	if len(groups) > 0 {
		var groupMax int
		if err := DB.Model(&model.AddressBookCollectionRule{}).
			Where("user_id = ? AND collection_id = ? AND type = ? AND to_id IN ?",
				ownerID, collectionID, model.ShareAddressBookRuleTypeGroup, groups).
			Select("COALESCE(MAX(rule), 0)").Scan(&groupMax).Error; err != nil {
			return 0
		}
		if groupMax > 0 {
			return groupMax
		}
	}
	var everyone model.AddressBookCollectionRule
	if err := DB.Where("user_id = ? AND collection_id = ? AND type = ? AND to_id = 0",
		ownerID, collectionID, model.ShareAddressBookRuleTypeAll).
		Order("id DESC").First(&everyone).Error; err == nil {
		return everyone.Rule
	}
	return 0
}

func (s *AddressBookService) CheckUserReadPrivilege(user *model.User, uid, cid uint) bool {
	return s.UserMaxRule(user, uid, cid) >= model.ShareAddressBookRuleRuleRead
}
func (s *AddressBookService) CheckUserWritePrivilege(user *model.User, uid, cid uint) bool {
	return s.UserMaxRule(user, uid, cid) >= model.ShareAddressBookRuleRuleReadWrite
}
func (s *AddressBookService) CheckUserFullControlPrivilege(user *model.User, uid, cid uint) bool {
	return s.UserMaxRule(user, uid, cid) >= model.ShareAddressBookRuleRuleFullControl
}

func (s *AddressBookService) CreateCollection(t *model.AddressBookCollection) error {
	return DB.Create(t).Error
}

func (s *AddressBookService) UpdateCollection(t *model.AddressBookCollection) error {
	return DB.Model(t).Updates(t).Error
}

func (s *AddressBookService) DeleteCollection(t *model.AddressBookCollection) error {
	if t == nil || t.Id == 0 { return errors.New("address book collection ID is required") }
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collection_id = ?", t.Id).
			Delete(&model.AddressBookCollectionRule{}).Error; err != nil { return err }
		// Deleting a collection also removes any previously archived entries
		// belonging to that collection, which cannot be restored afterwards.
		if err := tx.Unscoped().Where("collection_id = ?", t.Id).
			Delete(&model.AddressBook{}).Error; err != nil { return err }
		return tx.Delete(t).Error
	})
}

func (s *AddressBookService) RuleInfoById(u uint) *model.AddressBookCollectionRule {
	p := &model.AddressBookCollectionRule{}
	DB.Where("id = ?", u).First(p)
	return p
}
func (s *AddressBookService) RulePersonalInfoByToIdAndCid(toid, cid uint) *model.AddressBookCollectionRule {
	return s.RuleInfoByToIdAndCid(model.ShareAddressBookRuleTypePersonal, toid, cid)
}
func (s *AddressBookService) RuleInfoByToIdAndCid(t int, toid, cid uint) *model.AddressBookCollectionRule {
	p := &model.AddressBookCollectionRule{}
	DB.Where("type = ? and to_id = ? and collection_id = ?", t, toid, cid).First(p)
	return p
}

// RuleInfoByOwnerAndTarget resolves a sharing rule within its owner's address book.
func (s *AddressBookService) RuleInfoByOwnerAndTarget(ownerID uint, typ int, toID, collectionID uint) *model.AddressBookCollectionRule {
	rule := &model.AddressBookCollectionRule{}
	DB.Where("user_id = ? AND type = ? AND to_id = ? AND collection_id = ?",
		ownerID, typ, toID, collectionID).First(rule)
	return rule
}
func (s *AddressBookService) CreateRule(t *model.AddressBookCollectionRule) error {
	return DB.Create(t).Error
}

func (s *AddressBookService) ListRules(page uint, size uint, f func(tx *gorm.DB)) *model.AddressBookCollectionRuleList {
	res := &model.AddressBookCollectionRuleList{}
	res.Page = int64(page)
	res.PageSize = int64(size)
	tx := DB.Model(&model.AddressBookCollectionRule{})
	if f != nil {
		f(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, size))
	tx.Find(&res.AddressBookCollectionRule)
	return res
}

func (s *AddressBookService) UpdateRule(t *model.AddressBookCollectionRule) error {
	if t == nil || t.Id == 0 {
		return errors.New("address book sharing rule ID is required")
	}
	return DB.Model(&model.AddressBookCollectionRule{}).Where("id = ? AND user_id = ?", t.Id, t.UserId).
		Updates(map[string]interface{}{
			"collection_id": t.CollectionId, "type": t.Type, "to_id": t.ToId, "rule": t.Rule,
		}).Error
}

func (s *AddressBookService) DeleteRule(t *model.AddressBookCollectionRule) error {
	return DB.Delete(t).Error
}

// CheckCollectionOwner 检查Collection的所有者
func (s *AddressBookService) CheckCollectionOwner(uid uint, cid uint) bool {
	p := s.CollectionInfoById(cid)
	return p.UserId == uid
}

func (s *AddressBookService) BatchUpdateTags(abs []*model.AddressBook, tags []string) error {
	ids := make([]uint, 0)
	for _, ab := range abs {
		ids = append(ids, ab.RowId)
	}
	tagsv, _ := json.Marshal(tags)
	return DB.Model(&model.AddressBook{}).Where("row_id in ?", ids).Update("tags", tagsv).Error
}
