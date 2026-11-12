package service

import (
    "errors"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func ownedSessionDeviceIDs(userID uint) interface{} {
    return DB.Model(&model.Peer{}).Select("id").Where("user_id = ?",userID)
}

func ListOwnedActiveDeviceSessions(userID,page,size uint,sort ...string) ([]model.DeviceActiveConnection,int64,error) {
    rows:=[]model.DeviceActiveConnection{}
    var total int64
    q:=DB.Model(&model.DeviceActiveConnection{}).
        Where("device_id IN (?)",ownedSessionDeviceIDs(userID)).
        Where("last_seen_at >= ?",time.Now().Unix()-sessionAliveSeconds)
    if err:=q.Count(&total).Error;err!=nil{return nil,0,err}
    if err:=q.Order("last_seen_at DESC").Scopes(Paginate(page,size),SortScope(sort...)).Find(&rows).Error;err!=nil{return nil,0,err}
    return rows,total,nil
}

func ListOwnedDisconnectRequests(userID,page,size uint,sort ...string) ([]model.DeviceDisconnectRequest,int64,error) {
    rows:=[]model.DeviceDisconnectRequest{}
    var total int64
    q:=DB.Model(&model.DeviceDisconnectRequest{}).Where(
        "requested_by = ? OR device_id IN (?)",userID,ownedSessionDeviceIDs(userID))
    if err:=q.Count(&total).Error;err!=nil{return nil,0,err}
    if err:=q.Order("id DESC").Scopes(Paginate(page,size),SortScope(sort...)).Find(&rows).Error;err!=nil{return nil,0,err}
    return rows,total,nil
}

func RequestOwnedDeviceDisconnect(userID uint,deviceID string,connID int32) error {
    if userID==0 || deviceID=="" {return errors.New("请选择自己名下的设备")}
    var count int64
    if err:=DB.Model(&model.Peer{}).Where("user_id = ? AND id = ?",userID,deviceID).Count(&count).Error;err!=nil{return err}
    if count==0 {return errors.New("当前账户无权结束这台设备上的连接")}
    return RequestDeviceDisconnect(deviceID,connID,userID)
}
