package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"encoding/json"
)

// Diccionario estático para traducciones instantáneas de comandos y términos del CLI
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

// Mapeo de comandos en español hacia inglés para el CLI original
var mapeoComandos = map[string]string{
	"iniciar":   "start",
	"agentes":   "agents",
	"desplegar": "deploy",
	"ejecutar":  "run",
	"estado":    "status",
	"ayuda":     "--help",
}

// Estructura para respuesta de Google Translate API gratuita
type autoTranslateResponse [][]interface{}

// Función de traducción dinámica en caso de frases largas no presentes en el diccionario
func traducirConAPI(texto string) string {
	if strings.TrimSpace(texto) == "" {
		return texto
	}

	apiURL := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=en&tl=es&dt=t&q=%s", url.QueryEscape(texto))
	resp, err := http.Get(apiURL)
	if err != nil {
		return texto // Si no hay internet o falla la red, regresa el texto original
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return texto
	}

	var parsed autoTranslateResponse
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed) > 0 {
		var resultado strings.Builder
		for _, item := range parsed[0] {
			if len(item) > 0 {
				if str, ok := item[0].(string); ok {
					resultado.WriteString(str)
				}
			}
		}
		if resultado.Len() > 0 {
			return resultado.String()
		}
	}

	return texto
}

// Procesa y traduce cada línea enviada por el CLI
func procesarLinea(linea string) string {
	textoProcesado := linea

	// 1. Reemplazos rápidos del diccionario estático
	for en, es := range diccionarioEstatico {
		if strings.Contains(textoProcesado, en) {
			re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(en))
			textoProcesado = re.ReplaceAllString(textoProcesado, es)
		}
	}

	// 2. Si la línea cambió con el diccionario estático, devolverla de inmediato
	if textoProcesado != linea {
		return textoProcesado
	}

	// 3. Traducción dinámica para bloques de texto o explicaciones del agente
	if len(strings.TrimSpace(linea)) > 3 {
		return traducirConAPI(linea)
	}

	return linea
}

func main() {
	argsOriginales := os.Args[1:]
	argsTraducidos := make([]string, len(argsOriginales))

	// Traducir los comandos que el usuario escriba en español a inglés
	for i, arg := range argsOriginales {
		if trad, ok := mapeoComandos[strings.ToLower(arg)]; ok {
			argsTraducidos[i] = trad
		} else {
			argsTraducidos[i] = arg
		}
	}

	// Invocar el ejecutable nativo de Google Antigravity
	cmd := exec.Command("antigravity", argsTraducidos...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error al conectar stdout: %v\n", err)
		os.Exit(1)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error al conectar stderr: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error: No se encontró el ejecutable 'antigravity' en el sistema.\n")
		os.Exit(1)
	}

	// Escuchar stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(procesarLinea(scanner.Text()))
		}
	}()

	// Escuchar stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(os.Stderr, procesarLinea(scanner.Text()))
		}
	}()

	_ = cmd.Wait()
}
