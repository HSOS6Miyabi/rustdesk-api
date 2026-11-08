package service

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestDefaultAddressBookDoesNotChangeOtherCollections(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AddressBook{}); err != nil { t.Fatal(err) }
	users := seedGroups(t)
	AllService.AddressBookService = &AddressBookService{}
	uid := users["member"].Id
	saved := []model.AddressBook{
		{Id: "custom-device", UserId: uid, CollectionId: 20},
		{Id: "old-default", UserId: uid, CollectionId: 0},
	}
	for _, book := range saved {
		if err := DB.Create(&book).Error; err != nil { t.Fatal(err) }
	}
	input := []*model.AddressBook{{Id: "new-default"}}
	if err := AllService.AddressBookService.UpdateAddressBook(input, uid); err != nil {
		t.Fatal(err)
	}
	var otherCount int64
	if err := DB.Model(&model.AddressBook{}).
		Where("user_id = ? AND id = ?", uid, "custom-device").Count(&otherCount).Error; err != nil { t.Fatal(err) }
	if otherCount != 1 { t.Fatal("a different address book was changed") }
	var defaultCount int64
	if err := DB.Model(&model.AddressBook{}).
		Where("user_id = ? AND collection_id = 0", uid).Count(&defaultCount).Error; err != nil { t.Fatal(err) }
	if defaultCount != 1 { t.Fatalf("expected one default device, got %d", defaultCount) }
	err := AllService.AddressBookService.UpdateAddressBook([]*model.AddressBook{{Id:"duplicate"},{Id:"duplicate"}},uid)
	if err == nil { t.Fatal("duplicate IDs should fail") }
	var stillPresent int64
	if err := DB.Model(&model.AddressBook{}).
		Where("user_id = ? AND id = ?", uid, "new-default").Count(&stillPresent).Error; err != nil { t.Fatal(err) }
	if stillPresent != 1 { t.Fatal("transaction was not rolled back") }
}

func TestServerCommandReadsCompleteResponse(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil { received <- "listen error"; return }
		defer conn.Close()
		buffer := make([]byte, 512)
		n, _ := conn.Read(buffer)
		received <- string(buffer[:n])
		conn.Write([]byte("ready\n"))
		time.Sleep(50 * time.Millisecond)
		conn.Write([]byte("completed\n"))
	}()
	portText := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	port, err := strconv.Atoi(portText)
	if err != nil { t.Fatal(err) }
	result, err := (&ServerCmdService{}).SendSocketCmd("v4", port, "h")
	if err != nil { t.Fatal(err) }
	if <-received != "h" { t.Fatal("command was not delivered") }
	if result != "ready\ncompleted\n" { t.Fatalf("response truncated: %q", result) }
}
