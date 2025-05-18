package config_test

import (
	"os"
	"testing"

	"github.com/AlejandroHerr/go-common/pkg/logging"
	"github.com/AlejandroHerr/go-idasen-desk/internal/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	logger := logging.NewLogger()

	t.Run("uses default if files do not exist", func(t *testing.T) {
		t.Parallel()

		cfg, err := config.Load("nonexistent.yaml", logger)
		require.NoError(t, err, "should not error loading config")

		require.Equal(t, config.DefaultPort, cfg.Rest.Port, "should use default port")
		require.Equal(t, make([]string, 0), cfg.Rest.AuthTokens, "should be an empty array")
	})
	t.Run("uses default if file is empty", func(t *testing.T) {
		t.Parallel()

		cfg, err := config.Load("./__mock__/empty.yaml", logger)
		require.NoError(t, err, "should not error loading config")

		require.Equal(t, config.DefaultPort, cfg.Rest.Port, "should use default port")
		require.Equal(t, []string{}, cfg.Rest.AuthTokens, "should be an empty array")
	})
	t.Run("loads config from file", func(t *testing.T) {
		t.Parallel()

		file := "./__mock__/full.yaml"

		content, err := os.ReadFile(file)
		require.NoError(t, err)

		fileCfg := map[string](map[string]interface{}){
			"rest": map[string]interface{}{},
		}
		err = yaml.Unmarshal(content, &fileCfg)
		require.NoError(t, err)

		cfg, err := config.Load(file, logger)
		require.NoError(t, err, "should not error loading config")

		require.Equal(t, fileCfg["rest"]["port"], cfg.Rest.Port, "should use port from file")
		require.Equal(t, []string{"aaaaa", "bbbbb"}, cfg.Rest.AuthTokens, "should use tokens from file")
	})
}
