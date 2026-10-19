package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var diccionarioEstatico = map[string]string{
	"Usage:":                      "Uso:",
	"Options:":                    "Opciones:",
	"Commands:":                   "Comandos:",
	"Global Options:":             "Opciones Globales:",
	"Starting agent...":          "Iniciando agente...",
	"Agent running":               "Agente en ejecución",
	"Task completed successfully": "Tarea completada con éxito",
	"Failed to execute":           "Error al ejecutar",
	"Error:":                      "Error:",
	"Warning:":                    "Advertencia:",
	"Help:":                       "Ayuda:",
}

var mapeoComandos = map[string]string{
	"iniciar":   "start",
	"agentes":   "agents",
	"desplegar": "deploy",
	"ejecutar":  "run",
	"estado":    "status",
	"ayuda":     "--help",
}

func traducirConAPI(texto string) string {
	if strings.TrimSpace(texto) == "" {
		return texto
	}

	apiURL := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=en&tl=es&dt=t&q=%s", url.QueryEscape(texto))
	resp, err := http.Get(apiURL)
	if err != nil {
		return texto
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return texto
	}

	// Estructura segura para parsear el JSON de la API de traducción
	var parsed []interface{}
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed) > 0 {
		if sentences, ok := parsed[0].([]interface{}); ok {
			var resultado strings.Builder
			for _, s := range sentences {
				if sentenceParts, ok := s.([]interface{}); ok && len(sentenceParts) > 0 {
					if str, ok := sentenceParts[0].(string); ok {
						resultado.WriteString(str)
					}
				}
			}
			if resultado.Len() > 0 {
				return resultado.String()
			}
		}
	}

	return texto
}

func procesarLinea(linea string) string {
	textoProcesado := linea

	for en, es := range diccionarioEstatico {
		if strings.Contains(textoProcesado, en) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(en))
			textoProcesado = re.ReplaceAllString(textoProcesado, es)
		}
	}

	if textoProcesado != linea {
		return textoProcesado
	}

	if len(strings.TrimSpace(linea)) > 3 {
		return traducirConAPI(linea)
	}

	return linea
}

func main() {
	argsOriginales := os.Args[1:]
	argsTraducidos := make([]string, len(argsOriginales))

	for i, arg := range argsOriginales {
		if trad, ok := mapeoComandos[strings.ToLower(arg)]; ok {
			argsTraducidos[i] = trad
		} else {
			argsTraducidos[i] = arg
		}
	}

	cmd := exec.Command("antigravity", argsTraducidos...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Exit(1)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error: No se encontró el ejecutable 'antigravity' en el sistema.\n")
		os.Exit(1)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(procesarLinea(scanner.Text()))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, procesarLinea(scanner.Text()))
		}
	}()

	_ = cmd.Wait()
}
