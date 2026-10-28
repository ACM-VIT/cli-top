package cmd

import (
	"bufio"
	"cli-top/features"
	"cli-top/helpers"
	"cli-top/internal/contract"
	"cli-top/login"
	"cli-top/types"
	"fmt"
	"github.com/lpernett/godotenv"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func configValues() (map[string]string, error) {
	values, err := godotenv.Read(helpers.ConfigFilePath())
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, contract.Wrap("io_error", "Unable to read cli-top configuration.", err)
	}
	return values, nil
}
func writeConfig(values map[string]string) error {
	path := helpers.ConfigFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return contract.Wrap("io_error", "Unable to create configuration directory.", err)
	}
	text, err := godotenv.Marshal(values)
	if err != nil {
		return contract.Wrap("io_error", "Unable to encode configuration.", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".cli-top-*")
	if err != nil {
		return contract.Wrap("io_error", "Unable to write configuration.", err)
	}
	defer os.Remove(f.Name())
	if _, err = f.WriteString(text + "\n"); err != nil {
		f.Close()
		return contract.Wrap("io_error", "Unable to write configuration.", err)
	}
	if err = f.Close(); err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		return contract.Wrap("io_error", "Unable to save configuration.", err)
	}
	return nil
}
func authenticate() (features.Session, error) {
	values, err := configValues()
	if err != nil {
		return features.Session{}, err
	}
	get := func(k string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return values[k]
	}
	cookies := types.Cookies{SERVERID: get("SERVERID"), JSESSIONID: get("JSESSIONID"), CSRF: get("CSRF")}
	if helpers.ValidateCookies(cookies) {
		c, r, e := login.HomePageE(cookies)
		if e == nil && r != "" {
			return features.Session{Cookies: c, RegNo: r}, nil
		}
		if e != nil {
			ce := contract.From(e)
			if ce.Kind != "auth_required" && ce.Status != 404 {
				return features.Session{}, e
			}
		}
	}
	username, encrypted, key := get("VTOP_USERNAME"), get("PASSWORD"), get("KEY")
	if username == "" || encrypted == "" || key == "" {
		return features.Session{}, helpers.RequireLogin(false)
	}
	password, err := decryptPassword(encrypted, key)
	if err != nil {
		return features.Session{}, contract.New("auth_required", "Stored credentials are invalid; run cli-top login.")
	}
	cookies, err = login.LoginE(username, password)
	if err != nil {
		return features.Session{}, err
	}
	cookies, regNo, err := login.HomePageE(cookies)
	if err != nil {
		return features.Session{}, err
	}
	values["SERVERID"] = cookies.SERVERID
	values["JSESSIONID"] = cookies.JSESSIONID
	values["CSRF"] = cookies.CSRF
	values["REGNO"] = regNo
	if err = writeConfig(values); err != nil {
		return features.Session{}, err
	}
	return features.Session{Cookies: cookies, RegNo: regNo}, nil
}
func storeCredentials(username, password string, in io.Reader, out io.Writer) (any, error) {
	reader := bufio.NewReader(in)
	read := func(label string) string {
		fmt.Fprint(out, label)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line)
	}
	if helpers.CanPrompt() {
		if username == "" {
			username = read("Enter your username: ")
		}
		if password == "" {
			password = read("Enter your password (visible): ")
		}
	}
	if strings.TrimSpace(username) == "" || password == "" {
		e := contract.New("invalid_arguments", "Username and password are required.")
		e.RequiredFlags = []string{"--username", "--password"}
		return nil, e
	}
	key, err := GenerateAESKey()
	if err != nil {
		return nil, contract.Wrap("io_error", "Unable to initialize credential storage.", err)
	}
	encrypted, err := encryptPassword(password, key)
	if err != nil {
		return nil, contract.Wrap("io_error", "Unable to encrypt credentials.", err)
	}
	values, err := configValues()
	if err != nil {
		return nil, err
	}
	for _, k := range []string{"SERVERID", "CSRF", "JSESSIONID", "REGNO", "SEMESTER_CACHE"} {
		delete(values, k)
	}
	values["VTOP_USERNAME"] = strings.ToUpper(strings.TrimSpace(username))
	values["PASSWORD"] = encrypted
	values["KEY"] = key
	if err = writeConfig(values); err != nil {
		return nil, err
	}
	return types.OperationResult{Operation: "login", Status: "credentials_stored", Files: []types.FileResult{}}, nil
}
func logout() (any, error) {
	values, err := configValues()
	if err != nil {
		return nil, err
	}
	clean := map[string]string{}
	if values["UUID"] != "" {
		clean["UUID"] = values["UUID"]
	}
	if err = writeConfig(clean); err != nil {
		return nil, err
	}
	helpers.ForgetSemesterCache(values["REGNO"])
	return types.OperationResult{Operation: "logout", Status: "logged_out", Files: []types.FileResult{}}, nil
}
