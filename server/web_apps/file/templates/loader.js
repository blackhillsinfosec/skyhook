var {{ .Stage1KeyVar }}, int, algos_wasm, wasm_exec, wasm_helpers, wasm_worker, go;

async function xor(input, key, text){
    await new Blob([key]).arrayBuffer().then(buff => {key=new Uint8Array(buff)});

    let buff = new Uint8Array(input.length);
    for(let i=0; i<input.length; i++){
        buff[i] = input[i].charCodeAt(0);
    }
    input=null;

    for(let i=0; i<buff.byteLength; i++){
        buff[i] = buff[i]^key[i%key.byteLength];
    }

    if(text || text === undefined) {
        return new TextDecoder("utf8").decode(buff);
    }else{
        return buff;
    }
}

function checkHashLink() {
    let split = window.location.href.split('#');
    if (split.length === 2) {
        console.log("Key supplied via hash-link")
        {{ .Stage1KeyVar }}=split[1];
    }
}

checkHashLink();

int = setInterval(async () => {
    checkHashLink();
    if({{ .Stage1KeyVar }}) {
        console.log("Key retrieved")
        clearInterval(int);
        assemble({{ .Stage1KeyVar }});
    } else {
        console.log("Populate key var with XOR key...")
    }
}, 1000);

async function assemble(key){

    console.log("Assembling interface...");

    let h = document.getElementsByTagName("head")[0];
    let b = document.getElementsByTagName("body")[0];
    let ele = document.createElement("div");

    ele.setAttribute("id", "{{ .RootId }}");
    b.appendChild(ele);

    ele = document.createElement("meta")
    ele.setAttribute("charset", "UTF-8")
    h.appendChild(ele);

    ele = document.createElement("title");
    ele.innerText = "Skyhook";
    h.appendChild(ele);

    //======================
    // CSS & OTHER ARTIFACTS
    //======================

    const landing_urls = {
        "{{ .Urls.Favicon }}":["icon", "application/octet-stream"],
        "{{ .Urls.AssetManifest }}":["manifest", "text/json"],
        "{{ .Urls.BootstrapCss }}":["stylesheet", "text/css"],
        "{{ .Urls.MainCss }}":["stylesheet", "text/css"]
    };

    let url_keys = Object.keys(landing_urls);
    for (let i = 0; i < url_keys.length; i++) {
        let u_key = url_keys[i];
        let tup = landing_urls[u_key];

        await fetch(u_key + "?" + "{{ .QueryString }}")
            .then(async e => {
                    await e.blob()
                        .then(async b => {
                            await b.text()
                                .then(async t => {
                                    await xor(window.atob(t), key, true)
                                        .then(c => {
                                                let ele = document.createElement("link");
                                                ele.setAttribute("rel", tup[0]);
                                                ele.setAttribute("href", URL.createObjectURL(new Blob([c], {type: tup[1]})));
                                                h.appendChild(ele);
                                        })
                                })
                        })
                }
            )
    }

    //============================
    // LOAD EACH ENCRYPTED JS FILE
    //============================

    // A list of JS URLs that must be loaded
    // These values are set by server-side template logic
    const js_urls = {
        'wasm_exec': '{{ .Urls.WasmExecJs }}',
        'react_prod': '{{ .Urls.ReactProductionJs }}',
        'read_dom_prod': '{{ .Urls.ReactDomJs }}',
        'wasm_helpers': '{{ .Urls.WasmHelpersJs }}',
        'react_bootstrap': '{{ .Urls.ReactBootstrapJs }}',
        'main': '{{ .Urls.MainJs }}',
    };

    url_keys = Object.keys(js_urls);

    for (let i = 0; i < url_keys.length; i++) {

        let u_key = url_keys[i];

        if(u_key === 'react_bootstrap'){
            let s = document.createElement("script")
            s.setAttribute("src", js_urls[u_key])
            h.appendChild(s);
            continue
        }

        //===========================
        // RETRIEVE ENCRYPTED JS FILE
        //===========================

        await fetch(js_urls[u_key] + "?" + "{{ .QueryString }}")
            .then(async e => {
                await e.blob()
                    .then(async b => {
                        await b.text()
                            .then(async t => {
                                await xor(window.atob(t), key, true)
                                    .then(c => {
                                        if(u_key !== 'main') {
                                            eval(c);
                                            if(u_key === 'wasm_exec'){go = new Go()}
                                        } else {
                                            let s = document.createElement("script");
                                            s.setAttribute("defer", "defer");
                                            s.setAttribute('src', URL.createObjectURL(new Blob([c], {type: "text/javascript"})));
                                            h.appendChild(s);
                                        }
                                    })
                            })
                        })
            }
        )
    }

    const wasm_urls = {
        "algos_wasm": "{{ .Urls.AlgosWasmJs }}",
        "wasm_exec": "{{ .Urls.WasmExecJs }}",
        "wasm_helpers": "{{ .Urls.WasmHelpersJs }}",
        "wasm_worker": "{{ .Urls.WasmWorkerJs }}"
    };

    url_keys = Object.keys(wasm_urls);
    for(let i=0; i<url_keys.length; i++){
        let url = wasm_urls[url_keys[i]];
        await fetch(url + "?" + "{{ .QueryString }}" )
            .then(e => {
                e.blob()
                    .then(b => {
                        b.text()
                            .then(t => {
                                xor(window.atob(t), key, false)
                                    .then(async c => {
                                    if(url === wasm_urls["algos_wasm"]){
                                        await WebAssembly.instantiate(c.buffer, go.importObject)
                                            .then(r =>{go.run(r.instance);});
                                        algos_wasm=c.buffer;
                                    } else {
                                        await new Blob([c]).text()
                                            .then(t => {
                                                switch(url){
                                                    case wasm_urls["wasm_exec"]:
                                                        wasm_exec=t;
                                                        break;
                                                    case wasm_urls["wasm_helpers"]:
                                                        wasm_helpers=t;
                                                        break;
                                                    case wasm_urls["wasm_worker"]:
                                                        wasm_worker=window.URL.createObjectURL(new Blob([t]));
                                                }
                                        })
                                    }
                                })
                            })
                    })})
    }
}

console.log("Populate {{ .Stage1KeyVar }} JS variable to decrypt...")