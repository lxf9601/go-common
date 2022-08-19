package conf

// mysql配置项
type MySqlConf struct {
	Url             string // 连接字符串
	MaxIdleConns    int    // 最大空闲连接数
	MaxOpenConns    int    // 最大打开连接数
	ConnMaxLifetime int    // 连接生命周期
}

// pgsql配置项
type PgSqlConf struct {
	MySqlConf
}

// Redis配置项
type RedisConf struct {
	Addr      string
	Password  string
	Db        int
	KeyPrefix string
}
