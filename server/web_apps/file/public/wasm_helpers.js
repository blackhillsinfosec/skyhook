// WARNING: This is just a development file.
//
// The authoritative JS file is wasm_helpers.js in
// the skyhook-wasm repository.

async function checkBaseObfArgs(act, value){
    if(act !== "deobf" && act !== "obf"){
        throw "act must be 'obf' or 'deobf'"
    }
    if(typeof(value) === "string"){
        await skyStob(value).then((v) => {value=v});
    }
    return {act: act, value: value}
}

// Xor proxies xor encoding/decoding requests to the WASM module.
async function Xor(act, value, config){
    a = await checkBaseObfArgs(act, value);

    if(config.key === undefined){
        throw "key argument is required"
    } else if(typeof(config.key) === "string"){
        await skyStob(config.key).then((v) => {config.key=v});
    }

    let output;
    await skyXor(a.act, a.value, config.key).then((v) => {output=v});

    return output;
}

// B64 proxies base64 encoding/decoding requests to the WASM module.
async function B64(act, value, config){
    a = await checkBaseObfArgs(act, value);

    if(config.rounds === undefined){
        config.rounds=1;
    }else if(typeof(config.rounds) !== "number" || config.rounds < 1){
        throw "rounds must be a number"
    }

    let output;
    await skyB64(a.act, a.value, config.rounds).then((v) => {output=v});

    return output;
}

// Blowfish proxies xor encoding/decoding requests to the WASM module.
async function Aes(act, value, config){
    a = await checkBaseObfArgs(act, value);

    if(config.key === undefined){
        throw "key argument is required"
    } else if(typeof(config.key) === "string"){
        await skyStob(config.key).then((v) => {config.key=v});
    }

    let output;
    await skyAes(a.act, a.value, config.key, config.salt).then((v) => {output=v});

    return output;
}

// Blowfish proxies xor encoding/decoding requests to the WASM module.
async function Twofish(act, value, config){
    a = await checkBaseObfArgs(act, value);

    if(config.key === undefined){
        throw "key argument is required"
    } else if(typeof(config.key) === "string"){
        await skyStob(config.key).then((v) => {config.key=v});
    }

    let output;
    await skyTwofish(a.act, a.value, config.key, config.salt).then((v) => {output=v});

    return output;
}

// Blowfish proxies xor encoding/decoding requests to the WASM module.
async function Blowfish(act, value, config){
    a = await checkBaseObfArgs(act, value);

    if(config.key === undefined){
        throw "key argument is required"
    } else if(typeof(config.key) === "string"){
        await skyStob(config.key).then((v) => {config.key=v});
    }

    if(config.salt === undefined){
        throw "salt argument is required"
    } else if(typeof(config.salt) === "string"){
        await skyStob(config.salt).then((v) => {config.salt=v});
    }

    let output;
    await skyBlowfish(a.act, a.value, config.key, config.salt).then((v) => {output=v});

    return output;
}

// SkyObfuscators is an object mapping lowercase algorithm names to obfuscation
// functions.
const SkyObfuscators = {
    xor: Xor,
    base64: B64,
    blowfish: Blowfish,
    aes: Aes,
    twofish: Twofish,
}

// RunObfs executes obfuscation/deobfuscation functions in
// accordance with the current obfuscation configuration, which
// is an order-relevant list of obfuscation algorithm settings.
async function RunObfs(act, value, config, addtl){

    // Preserve the current configuration by copying it.
    let conf = [...config];

    if(act === "deobf") {
        // Deobfuscation requires processing the obfuscation chain
        // in the reverse order.
        conf.reverse();
        // Perform initial base64 decoding.
        await skyB64("deobf", value, 1).then((v) => {value=v})
    }

    // Apply each algorithm.
    for(let i=0; i<conf.length; i++){
        let algo = SkyObfuscators[conf[i].algo];
        if(algo === undefined){
            throw(`Invalid obfuscator algorithm specified: ${conf[i].algo}`);
        }
        await algo(act, value, conf[i].config).then((out) => {value=out});
    }

    if(act === "obf"){
        // Obfuscated values will always receive a single round of base64 encoding
        // to avoid breaking HTTP.
        await skyB64("obf", value, 1).then((v) => {value=v})
    }

    // Return the manipulated value.
    return value
}