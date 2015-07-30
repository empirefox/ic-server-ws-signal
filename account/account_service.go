package account

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	. "github.com/empirefox/ic-server-ws-signal/gorm"
)

var (
	aservice            = NewAccountService()
	AccountNotAuthedErr = errors.New("Account is not authed")
)

func SetService(a AccountService) {
	if a == nil {
		aservice = NewAccountService()
	} else {
		aservice = a
	}
}

// Used by application
type AccountService interface {
	CreateTables() error
	DropTables() error

	FindOauthProviders(ops *[]OauthProvider) error
	SaveOauthProvider(op *OauthProvider) error

	OnOid(o *Oauth, provider, oid string) error
	Permitted(o *Oauth, c *gin.Context) bool
	Valid(o *Oauth) bool

	GetOnes(a *Account) error
	RegOne(a *Account, o *One) error
	ViewOne(a *Account, o *One) error
	RemoveOne(a *Account, o *One) error
	Logoff(a *Account) error

	FindOne(o *One, addr []byte) error
	FindOneIfOwner(o *One, id, ownerId uint) error
	Save(o *One) error
	Viewers(o *One) error
}

func NewAccountService() AccountService {
	return accountService{}
}

type accountService struct{}

func (accountService) CreateTables() error {
	ao := &AccountOne{}
	one := &One{}
	oauth := &Oauth{}
	return DB.CreateTable(ao).CreateTable(&Account{}).CreateTable(one).
		CreateTable(oauth).CreateTable(&OauthProvider{}).
		Model(ao).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE").
		Model(ao).AddForeignKey("one_id", "ones", "CASCADE", "CASCADE").
		Model(one).AddForeignKey("owner_id", "accounts", "CASCADE", "CASCADE").
		Model(oauth).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE").Error
}

func (accountService) DropTables() error {
	return DB.DropTableIfExists(&AccountOne{}).DropTableIfExists(&Oauth{}).DropTableIfExists(&One{}).
		DropTableIfExists(&Account{}).DropTableIfExists(&OauthProvider{}).Error
}

func (accountService) FindOauthProviders(ops *[]OauthProvider) error {
	return DB.Where(OauthProvider{Enabled: true}).Find(ops).Error
}

func (accountService) SaveOauthProvider(op *OauthProvider) error {
	return DB.Save(op).Error
}

func (accountService) Logoff(a *Account) error {
	return DB.Unscoped().Delete(a).Error
}

func (accountService) GetOnes(a *Account) error {
	return DB.Model(a).Association("Ones").Find(&a.Ones).Error
}

// one must be non-exist record
// a   must be from Oauth.OnOid
func (accountService) RegOne(a *Account, one *One) error {
	one.Owner = *a
	return a.ViewOne(one)
}

// one must be exist record
// a   must be from Oauth.OnOid
func (accountService) ViewOne(a *Account, one *One) error {
	return DB.Model(a).Association("Ones").Append(one).Error
}

func indexOf(a *Account, one *One) int {
	for i := range a.Ones {
		if a.Ones[i].ID == one.ID {
			return i
		}
	}
	return -1
}

// one must be exist record
// a   must be from Oauth.OnOid
func (accountService) RemoveOne(a *Account, one *One) error {
	if one.OwnerId == a.ID {
		err := DB.Delete(one).Error
		if err != nil {
			return err
		}
		if i := indexOf(a, one); i != -1 {
			a.Ones = append(a.Ones[:i], a.Ones[i+1:]...)
		}
		return nil
	}
	return DB.Model(a).Association("Ones").Delete(one).Error
}

func (accountService) FindOne(o *One, addr []byte) error {
	var w One
	w.SecretAddress = string(addr)
	err := DB.Where(w).Preload("Owner").First(o).Error
	if err != nil {
		return err
	}
	err = DB.Model(o).Related(&o.Accounts, "Accounts").Error
	if err == gorm.RecordNotFound {
		return nil
	}
	return err
}

func (accountService) FindOneIfOwner(o *One, id, ownerId uint) error {
	var w One
	w.ID = id
	w.OwnerId = ownerId
	return DB.Where(w).First(o).Error
}

func (accountService) Save(o *One) error {
	return DB.Save(o).Error
}

func (accountService) Viewers(o *One) error {
	return DB.Model(o).Association("Accounts").Find(&o.Accounts).Error
}

func (accountService) OnOid(o *Oauth, provider, oid string) error {
	err := DB.Where(Oauth{Provider: provider, Oid: oid, Validated: true, Enabled: true}).
		Attrs(Oauth{Account: Account{BaseModel: BaseModel{Name: provider + oid}, Enabled: true}}).
		Preload("Account").FirstOrCreate(o).Error
	return err
}

func (accountService) Permitted(o *Oauth, c *gin.Context) bool { return o.Validated }

func (accountService) Valid(o *Oauth) bool { return o.Enabled && o.Account.Enabled }
