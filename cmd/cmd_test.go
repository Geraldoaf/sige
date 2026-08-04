package cmd

import (
	"os"
	"testing"
)

func TestCLIInitConfig(t *testing.T) {

	_ = os.Remove("config.json")
	defer os.Remove("config.json")

	initConfigCmd.Run(initConfigCmd, []string{})

	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		t.Fatalf("config.json não foi gerado pelo comando init-config")
	}

	data, err := os.ReadFile("config.json")
	if err != nil {
		t.Fatalf("Erro ao ler config.json gerado: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("config.json gerado está vazio")
	}
}

func TestCLIVerifyCgroups(t *testing.T) {

	verifyCgroupsCmd.Run(verifyCgroupsCmd, []string{})
}

func TestCLIRootCmdFlags(t *testing.T) {
	if rootCmd.Use != "sige" {
		t.Errorf("rootCmd.Use incorreto: esperado 'sige', obtido '%s'", rootCmd.Use)
	}
	if rootCmd.Short != "SIGE - Sistema de Isolamento e Gerenciamento de Execução" {
		t.Errorf("rootCmd.Short incorreto: obtido '%s'", rootCmd.Short)
	}
}

func TestCLIRunTaskFlags(t *testing.T) {
	if runTaskCmd.Use != "run-task [command]" {
		t.Errorf("runTaskCmd.Use incorreto: %s", runTaskCmd.Use)
	}

	memFlag := runTaskCmd.Flags().Lookup("mem")
	if memFlag == nil {
		t.Errorf("Flag 'mem' não encontrada no comando run-task")
	}

	cpuFlag := runTaskCmd.Flags().Lookup("cpu")
	if cpuFlag == nil {
		t.Errorf("Flag 'cpu' não encontrada no comando run-task")
	}

	timeoutFlag := runTaskCmd.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Errorf("Flag 'timeout' não encontrada no comando run-task")
	}

	tmpFlag := runTaskCmd.Flags().Lookup("tmp-limit")
	if tmpFlag == nil {
		t.Errorf("Flag 'tmp-limit' não encontrada no comando run-task")
	}

	fileLimitFlag := runTaskCmd.Flags().Lookup("file-limit")
	if fileLimitFlag == nil {
		t.Errorf("Flag 'file-limit' não encontrada no comando run-task")
	}

	nofileLimitFlag := runTaskCmd.Flags().Lookup("nofile-limit")
	if nofileLimitFlag == nil {
		t.Errorf("Flag 'nofile-limit' não encontrada no comando run-task")
	}
}
