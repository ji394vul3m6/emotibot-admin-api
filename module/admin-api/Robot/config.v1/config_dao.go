package config

import (
	"database/sql"
	"time"

	"emotibot.com/emotigo/module/admin-api/util"
)

const moduleBFSource = "bf-env"

type configDaoInterface interface {
	GetDefaultConfigs() ([]*Config, error)
	GetConfigs(appid string) ([]*Config, error)
	GetConfig(appid, configName string) (*Config, error)
	SetConfig(appid, module, configName, value string) error
	SetConfigToDefault(appid, configName string) error
}

type configMySQL struct {
	db *sql.DB
}

func (dao *configMySQL) CheckDB() bool {
	if dao.db == nil {
		dao.db = util.GetMainDB()
	}
	return dao.db != nil
}

func (dao configMySQL) GetDefaultConfigs() ([]*Config, error) {
	var err error
	defer func() {
		util.ShowError(err)
	}()
	if !dao.CheckDB() {
		return nil, util.ErrDBNotInit
	}

	queryStr := `
		SELECT code, module, value, update_time
		FROM bfop_config
		WHERE appid = ''`
	rows, err := dao.db.Query(queryStr)
	if err != nil {
		return nil, err
	}

	ret := []*Config{}
	for rows.Next() {
		t := Config{}
		err = rows.Scan(&t.Code, &t.Module, &t.Value, &t.UpdateTime)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &t)
	}
	rows.Close()

	// Get configs from BF system
	queryStr = "SELECT `name`, `value` FROM `ent_config` WHERE `module` not in ('helper', 'validator', 'functions')"
	rows, err = dao.db.Query(queryStr)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		t := Config{}
		err = rows.Scan(&t.Code, &t.Value)
		if err != nil {
			return nil, err
		}
		t.Module = moduleBFSource
		ret = append(ret, &t)
	}
	rows.Close()

	return ret, nil
}

func (dao configMySQL) GetConfigs(appid string) ([]*Config, error) {
	var err error
	defer func() {
		util.ShowError(err)
	}()
	if !dao.CheckDB() {
		return nil, util.ErrDBNotInit
	}

	queryStr := `
		SELECT appid, code, module, value, update_time
		FROM bfop_config
		WHERE appid = ? OR appid = ''`
	rows, err := dao.db.Query(queryStr, appid)
	if err != nil {
		return nil, err
	}

	configMap := map[string]*Config{}
	for rows.Next() {
		t := Config{}
		robot := ""
		err = rows.Scan(&robot, &t.Code, &t.Module, &t.Value, &t.UpdateTime)
		if err != nil {
			return nil, err
		}
		if _, ok := configMap[robot]; !ok {
			// If config is not set, set it to map
			configMap[t.Code] = &t
		} else if robot != "" {
			// If config is set, but current row is the custom value of robot, change its value.
			configMap[t.Code] = &t
		}
	}
	rows.Close()

	ret := []*Config{}
	for _, c := range configMap {
		ret = append(ret, c)
	}

	// Get configs from BF system
	queryStr =
		"SELECT default.name, default.value, custom.value FROM " +
			"(SELECT `name`, `value` FROM `ent_config` WHERE `module` not in ('helper', 'validator', 'functions')) AS `default` " +
			"LEFT JOIN " +
			"(SELECT `name`, `app_id`, `value` FROM `ent_config_appid_customization` WHERE `app_id` = ?) AS `custom` " +
			"ON default.name = custom.name"

	rows, err = dao.db.Query(queryStr, appid)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		t := Config{}
		var customValue *string
		err = rows.Scan(&t.Code, &t.Value, &customValue)
		if err != nil {
			return nil, err
		}
		if customValue != nil {
			t.Value = *customValue
		}
		t.Module = moduleBFSource
		ret = append(ret, &t)
	}
	rows.Close()
	return ret, nil
}

func (dao configMySQL) GetConfig(appid, configName string) (*Config, error) {
	var err error
	defer func() {
		util.ShowError(err)
	}()
	if !dao.CheckDB() {
		return nil, util.ErrDBNotInit
	}

	// order by appid will let row with appid nonempty be the first row.
	queryStr := `
		SELECT module, value, update_time
		FROM bfop_config
		WHERE
			(appid = ? OR appid = '') AND
			code = ?
		ORDER BY appid DESC limit 1`
	row := dao.db.QueryRow(queryStr, appid, configName)

	ret := Config{}
	err = row.Scan(&ret.Code, &ret.Module, &ret.Value, &ret.UpdateTime)
	if err == nil {
		return &ret, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// If config not find in BFOP system, Get config from BF system
	queryStr =
		"SELECT default.name, default.value, custom.value FROM " +
			"(SELECT `name`, `value` FROM `ent_config` WHERE name = ?) AS `default` " +
			"LEFT JOIN " +
			"(SELECT `name`, `app_id`, `value` FROM `ent_config_appid_customization` WHERE `app_id` = ?) AS `custom` " +
			"ON default.name = custom.name"
	row = dao.db.QueryRow(queryStr, configName, appid)

	var customValue *string
	err = row.Scan(&ret.Code, &ret.Value, &customValue)
	if err != nil {
		return nil, err
	}

	if customValue != nil {
		ret.Value = *customValue
	}
	ret.Module = moduleBFSource
	return &ret, nil
}

func (dao configMySQL) SetConfig(appid, module, configName, value string) error {
	var err error
	defer func() {
		util.ShowError(err)
	}()
	if !dao.CheckDB() {
		return util.ErrDBNotInit
	}

	if module == moduleBFSource {
		queryStr := `
			INSERT INTO ent_config_appid_customization
			(name, app_id, value) VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE value = ?`
		_, err = dao.db.Exec(queryStr,
			configName, appid, value, value)
		if err != nil {
			return err
		}
	} else {
		now := time.Now()
		queryStr := `
			INSERT INTO bfop_config
			(appid, code, module, value, update_time) VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE value = ?, update_time = ?`
		_, err = dao.db.Exec(queryStr,
			appid, configName, module, value, now.Unix(), value, now.Unix())
		if err != nil {
			return err
		}
	}
	return nil
}

func (dao configMySQL) SetConfigToDefault(appid, configName string) error {
	var err error
	defer func() {
		util.ShowError(err)
	}()
	if !dao.CheckDB() {
		return util.ErrDBNotInit
	}

	now := time.Now()
	queryStr := `
	UPDATE bfop_config as config,
		(SELECT value FROM bfop_config WHERE appid = '' AND code = ?) as d
	SET
		config.value = d.value,
		config.update_time = ?
	WHERE
		config.appid = ? AND code = ?
	`
	_, err = dao.db.Exec(queryStr, configName, now.Unix(), appid, configName)
	if err != nil {
		return err
	}

	return nil
}
