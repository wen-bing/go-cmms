package shared

type User struct {
	ID          int     `db:"id"`
	Username    string  `db:"username"`
	Name        string  `db:"name"`
	Address     string  `db:"address"`
	Passwd      string  `db:"passwd"`
	Email       string  `db:"email"`
	Role        string  `db:"role"`
	SMS         string  `db:"sms"`
	HourlyRate  float64 `db:"hourly_rate"`
	SiteID      int     `db:"site_id"`
	Notes       string  `db:"notes"`
	Sites       []Site  `db:"site"`
	UseMobile   bool    `db:"use_mobile"`
	Local       bool    `db:"local"`
	IsTech      bool    `db:"is_tech"`
	CanAllocate bool    `db:"can_allocate"`
}

type UserRPCData struct {
	Channel int
	ID      int
	User    *User
}

type UserUpdate struct {
	Channel  int    `db:"channel"`
	ID       int    `db:"id"`
	Username string `db:"username"`
	Name     string `db:"name"`
	Passwd   string `db:"passwd"`
	Email    string `db:"email"`
	SMS      string `db:"sms"`
}

type UserSite struct {
	SiteID    int    `db:"site_id"`
	SiteName  string `db:"site_name"`
	Count     int    `db:"count"`
	Highlight *bool  `db:"highlight"`
}

type SiteUser struct {
	UserID   int    `db:"user_id"`
	Username string `db:"username"`
	Count    int    `db:"count"`
}

type UserSiteRequest struct {
	Channel int
	ID      int
	User    *User
	Site    *Site
}

type UserSiteSetRequest struct {
	Channel int
	UserID  int
	SiteID  int
	Role    string
	IsSet   bool
}

type UserOnline struct {
	ID          int      `db:"id"`
	Username    string   `db:"username"`
	Browser     string   `db:"browser"`
	IP          string   `db:"ip"`
	Name        string   `db:"name"`
	Email       string   `db:"email"`
	Role        string   `db:"role"`
	SMS         string   `db:"sms"`
	IsTech      bool     `db:"is_tech"`
	CanAllocate bool     `db:"can_allocate"`
	Route       string   `db:"route"`
	Routes      []string `db:"routes"`
	Duration    string   `db:"duration"`
	Channel     int      `db:"channel"`
}
