package task

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/869413421/wechatbot/app/config"
)

var (
	db     *gorm.DB
	dbOnce sync.Once
)

// InitDatabase 初始化数据库连接
func InitDatabase() error {
	var initErr error
	dbOnce.Do(func() {
		cfg := config.LoadConfig()
		mysqlCfg := cfg.MySQL

		// 设置默认值
		if mysqlCfg.Host == "" {
			mysqlCfg.Host = "localhost"
		}
		if mysqlCfg.Port == 0 {
			mysqlCfg.Port = 3306
		}
		if mysqlCfg.User == "" {
			mysqlCfg.User = "root"
		}
		if mysqlCfg.Database == "" {
			mysqlCfg.Database = "wechatbot_tasks"
		}
		if mysqlCfg.Charset == "" {
			mysqlCfg.Charset = "utf8mb4"
		}

		log.Printf("Connecting to MySQL database: %s@%s:%d/%s\n", mysqlCfg.User, mysqlCfg.Host, mysqlCfg.Port, mysqlCfg.Database)

		// 先连接到MySQL服务器（不指定数据库）以创建数据库
		dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
			mysqlCfg.User,
			mysqlCfg.Password,
			mysqlCfg.Host,
			mysqlCfg.Port,
			mysqlCfg.Charset,
		)

		tempDB, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			initErr = fmt.Errorf("failed to connect to MySQL server: %v", err)
			return
		}

		// 创建数据库（如果不存在）
		createDBQuery := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", mysqlCfg.Database)
		if err := tempDB.Exec(createDBQuery).Error; err != nil {
			initErr = fmt.Errorf("failed to create database: %v", err)
			return
		}
		log.Printf("Database '%s' created or already exists\n", mysqlCfg.Database)

		// 连接到指定数据库
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			mysqlCfg.User,
			mysqlCfg.Password,
			mysqlCfg.Host,
			mysqlCfg.Port,
			mysqlCfg.Database,
			mysqlCfg.Charset,
		)

		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			initErr = fmt.Errorf("failed to connect to database: %v", err)
			return
		}

		// 获取底层sql.DB设置连接池
		sqlDB, err := db.DB()
		if err != nil {
			initErr = fmt.Errorf("failed to get sql.DB: %v", err)
			return
		}
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)

		log.Printf("Successfully connected to MySQL database\n")

		// 自动迁移表结构
		if err := autoMigrate(); err != nil {
			initErr = fmt.Errorf("failed to migrate tables: %v", err)
			return
		}
	})

	return initErr
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate() error {
	// 迁移Task模型
	if err := db.AutoMigrate(&Task{}); err != nil {
		return fmt.Errorf("failed to migrate tasks table: %v", err)
	}
	log.Printf("Tasks table migrated\n")

	// 迁移TaskDependency模型
	if err := db.AutoMigrate(&TaskDependency{}); err != nil {
		return fmt.Errorf("failed to migrate task_dependencies table: %v", err)
	}
	log.Printf("Task dependencies table migrated\n")

	return nil
}

// GetDB 获取GORM数据库连接
func GetDB() *gorm.DB {
	return db
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

