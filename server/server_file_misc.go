package server

import (
    "bytes"
    "fmt"
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/tdewolff/minify"
    "github.com/tdewolff/minify/js"
    "golang.org/x/exp/slices"
    "io"
    "math/rand"
    "path"
    "strings"
    "text/template"
    "time"
)

var (
    staticPaths = []string{
        "web_apps/file/build/static/css/main.css",
        "web_apps/file/build/static/css/main.css.map",
        "web_apps/file/build/static/js/chunk.js",
        "web_apps/file/build/static/js/chunk.js.map",
        "web_apps/file/build/static/js/main.js",
        "web_apps/file/build/static/js/main.js.map",
        "web_apps/file/build/static/js/main.js.LICENSE.txt",
    }
    landingPrefix    = "web_apps/file/build"
    ascii            asciiSubCipher
    jsLoaderTemplS0  *template.Template
    jsLoaderTemplS1  *template.Template
    htmlLoaderTemp   *template.Template
    jsLoaderTempUrls jsLoaderTemplateUrls
    jsMinifier       = minify.New()
)

func init() {
    rand.Seed(time.Now().UnixNano())
    ascii = newAsciiSubCipher(false)
    jsMinifier.AddFunc("application/javascript", js.Minify)
    var err error
    for t, v := range map[**template.Template]string{
        &jsLoaderTemplS0: "web_apps/file/templates/loader_template.js",
        &jsLoaderTemplS1: "web_apps/file/templates/loader.js",
        &htmlLoaderTemp:  "web_apps/file/templates/loader.html"} {
        if *t, err = template.ParseFS(FileSpa, v); err != nil {
            panic(err)
        }
    }
}

func lookupStatic(s string) string {
    for _, p := range staticPaths {
        if strings.HasSuffix(p, s) {
            return p
        }
    }
    return path.Join(landingPrefix, s)
}

// parseFileExt accepts a file name and parses the final
// ".part" and returns it. An empty string is returned if
// no file extension is extracted.
func parseFileExt(fname string) (ext string) {
    s := strings.Split(fname, ".")
    if len(s) > 1 {
        ext = s[len(s)-1]
    }
    return ext
}

// randOverOne returns an integer 1>=max.
func randOverOne(max int) int {
    var n int
    for ; n == 0; n = rand.Intn(max) {
    }
    return n
}

// randOverOneMany returns many integers unique 1>=max.
func randOverOneMany(count, max int) (out []int) {
    for n := 0; n < count; n++ {
        v := randOverOne(max)
        if slices.Contains(out, v) {
            n--
            continue
        }
        out = append(out, v)
    }
    return out
}

// doMinifi is a shortcut to minify JS code, resulting in
// coments being stripped and mild code alteration.
func doMinify(d []byte) []byte {
    buff := make([]byte, 0)
    w := bytes.NewBuffer(buff)
    jsMinifier.Minify("application/javascript", w, bytes.NewReader(d))
    buff, _ = io.ReadAll(w)
    return buff
}

type loaderHtmlValues struct {
    Script string
}

// jsLoaderTemplateUrls is a mapping of friendly names to
// fake URLs for use by the loader template.
type jsLoaderTemplateUrls struct {
    Favicon           string
    AssetManifest     string
    AlgosWasmJs       string
    BootstrapJs       string
    BootstrapCss      string
    ReactBootstrapJs  string
    ReactDomJs        string
    ReactProductionJs string
    MainCss           string
    MainJs            string
    WasmExecJs        string
    WasmHelpersJs     string
    WasmVarsJs        string
    WasmWorkerJs      string
}

func (l *jsLoaderTemplateUrls) Insert(realPath, fakePath string) {
    _, realName := path.Split(realPath)
    switch realName {
    case "algos.wasm":
        jsLoaderTempUrls.AlgosWasmJs = fakePath
    case "asset-manifest.json":
        jsLoaderTempUrls.AssetManifest = fakePath
    case "bootstrap.min.js":
        jsLoaderTempUrls.BootstrapJs = fakePath
    case "bootstrap.min.css":
        jsLoaderTempUrls.BootstrapCss = fakePath
    case "favicon.ico":
        jsLoaderTempUrls.Favicon = fakePath
    case "react-bootstrap.min.js":
        jsLoaderTempUrls.ReactBootstrapJs = fakePath
    case "react.production.min.js":
        jsLoaderTempUrls.ReactProductionJs = fakePath
    case "react-dom.production.min.js":
        jsLoaderTempUrls.ReactDomJs = fakePath
    case "main.js":
        jsLoaderTempUrls.MainJs = fakePath
    case "main.css":
        jsLoaderTempUrls.MainCss = fakePath
    case "wasm_exec.js":
        jsLoaderTempUrls.WasmExecJs = fakePath
    case "wasm_helpers.js":
        jsLoaderTempUrls.WasmHelpersJs = fakePath
    case "wasm_vars.js":
        jsLoaderTempUrls.WasmVarsJs = fakePath
    case "wasm_worker.js":
        jsLoaderTempUrls.WasmWorkerJs = fakePath
    }
}

// loaderTemplateValues provides base values for rendering
// the encrypted loader template.
type loaderTemplateValues struct {
    Stage0KeyVar string
    Stage0Key    string
    PayVar       string
    Pay          string
    BuffVar      string
    RootId       string
    QueryString  string
    Stage1KeyVar string
    Urls         jsLoaderTemplateUrls
}

// newAsciiSubCipher initializes and returns an asciiSubCipher object.
//
// Setting shuf to true results in pseudorandomization of the underlying
// ascii values.
func newAsciiSubCipher(shuf bool) asciiSubCipher {
    a := asciiSubCipher{}
    for i := 32; i < 127; i++ {
        a = append(a, byte(i))
    }
    a = append(a, []byte{0x09, 0x0a, 0x0d}...)
    if shuf {
        rand.Shuffle(len(a), func(i, j int) {
            a[i], a[j] = a[j], a[i]
        })
    }
    return a
}

// asciiSubCipher provides a SubCrypt method that applies a substitution
// cipher to a series of ascii bytes that fall between the range of
// 32-127 with a distinct asciiSubCipher object that has been "shuffled".
//
// Use newAsciiSubCipher to create asciiSubCipher objects. Supply true
// to the shuf parameter to randomize the underlying values.
type asciiSubCipher []byte

// SubCrypt applies an inline substitution cipher to payload.
//
// minKey is a pointer to a map that will hold a mapping of substituted values
// to real values. A panic occurs should minKey be any type other than:
//
// - nil - which disregards the key
// - *map[string]string
// - *map[byte]byte
func (a asciiSubCipher) SubCrypt(payload []byte, key *asciiSubCipher, minKey any) {
    for i := 0; i < len(payload); i++ {

        //=========================
        // PERFORM THE SUBSTITUTION
        //=========================

        aInd := slices.Index(a, payload[i])
        sub := (*key)[aInd]
        payload[i] = sub

        //===============
        // UPDATE THE KEY
        //===============

        if minKey != nil {
            switch v := minKey.(type) {
            case *map[string]string:
                (*v)[string(sub)] = string(a[aInd])
            case *map[byte]byte:
                (*v)[sub] = a[aInd]
            default:
                panic("key to ascii.SubCrypt must be map[byte]byte or map[string]string")
            }
        }
    }
}

type assetManifest struct {
    Files       map[string]string `json:"files"`
    Entrypoints []string          `json:"entrypoints"`
}

type landingFileError string

func (err landingFileError) Error() string {
    return string(err)
}
func newLandingFileError(s string) landingFileError {
    return landingFileError(s)
}

type landingFileFields struct {
    Name    string
    Path    string
    Content []byte
}

type landingFile struct {
    Real landingFileFields
    Alt  landingFileFields
}

type landingFiles []landingFile

func (l *landingFiles) Get(pth string) (landingFile, error) {
    for _, f := range *l {
        if f.Real.Path == pth || f.Alt.Path == pth {
            return f, nil
        }
    }
    return landingFile{}, newLandingFileError(fmt.Sprintf("path not found: %v", pth))
}

func (l *landingFiles) Append(realPath, fakePath string, content []byte, xor *obfuscate.XOR) error {
    if l.Exists(realPath, fakePath) {
        return newLandingFileError(fmt.Sprintf("one of the paths already exist: %s -or- %s", realPath, fakePath))
    }
    _, rName := path.Split(realPath)
    _, fName := path.Split(fakePath)
    *l = append(*l, landingFile{
        Real: landingFileFields{
            Name:    rName,
            Path:    realPath,
            Content: content,
        },
        Alt: landingFileFields{
            Name: fName,
            Path: fakePath,
            Content: func() []byte {
                x, _ := xor.Obfuscate(content)
                return obfuscate.Base64Encode(x)
            }(),
        }})
    return nil
}

func (l *landingFiles) Exists(paths ...string) bool {
    for _, pth := range paths {
        for _, f := range *l {
            if f.Real.Path == pth || f.Alt.Path == pth {
                return true
            }
        }
    }
    return false
}

func extMimetype(ext string) string {
    switch ext {
    default:
        return ""
    case "txt":
        return "text"
    case "html":
        return "text/html"
    case "js":
        return "text/javascript"
    case "json", "map":
        return "application/json"
    case "png":
        return "image/png"
    case "wasm":
        return "application/wasm"
    }
}

func parseFileMimetype(fName string) string {
    return extMimetype(parseFileExt(fName))
}

func parseFilePathMimetype(p string) string {
    _, fName := path.Split(p)
    return parseFileMimetype(fName)
}
