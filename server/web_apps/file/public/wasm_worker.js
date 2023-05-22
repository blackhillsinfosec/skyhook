//======
// NOTES
//======
/*

worker.postMessage({

    // Text content of various WASM files.
    // - These are evaluated upon worker load.
    // - Makes WASM and supporting available to the web worker
    //   without having to download them from the server during
    //   initialization.

    wasm_exec: wasm_exec,
    algos_wasm: algos_wasm,
    wasm_helpers: wasm_helpers,

    // WASM function to execute.
    func: "RunObfs",

    // Arguments passed to the WASM function.
    args: ["deobf", data, obfs_config],

    // Determines which inputs should be converted to a Uint8Array
    // before running func.
    // - Note: Integers map back to arg offsets in array.
    bytefi_in: [1],

    // Flags indicating how to handle output.
    // - Note: When jsonify_out is true, stringify_out is implicitly
    //         called.

    stringify_out: false,
    jsonify_out: true
})

*/

onmessage = async (event) => {

    function wait(time) {
        return new Promise((res)=>setTimeout(res,time));
    }

    //==========
    // LOAD WASM
    //==========

    // Glue
    eval(event.data.wasm_exec);

    // WASM functions
    const go = new Go();
    await WebAssembly.instantiate(event.data.algos_wasm, go.importObject)
        .then((r) => {
            go.run(r.instance)
        });

    // Load wasm_helpers
    eval(event.data.wasm_helpers);

    //=====================================
    // CONVERT SPECIFIED ARGS TO BYTE ARRAY
    //=====================================
    // - Integers defined in bytefi_in map back to
    //   the offset in args.
    // - Each of them are converted in place to Uint8Array

    let input_conv_failure=null;
    if(event.data.bytefi_in){
        for(let i=0; i<event.data.bytefi_in.length; i++){
            let off=event.data.bytefi_in[i];
            if(typeof event.data.args[off] !== "string") { continue }
            await skyStob(event.data.args[off])
                .then((d) => {
                    event.data.args[off]=d;
                })
                .catch((e) => {
                    input_conv_failure=`Failed to bytefi input: ${e.message}`
                })
        }
    }

    //==================
    // RUN WASM FUNCTION
    //==================

    // Run the target function
        // Passing the function name to eval effectively
        // returns a reference to the function, which is
        // then called directly.
    let output_conv_failure=null;
    let output = await eval(event.data.func)(...event.data.args).then(async (d) => {

        //==============================
        // CONVERT OUTPUT TO STRING/JSON
        //==============================
        // - When stringify_out is true, then the output will be converted
        //   to a string.
        // - When jsonify_out is true, then the output will be converted
        //   to a string and parsed to JSON.
        //   - There is _no need_ to enable stringify_out when using
        //     jsonify_out as it will be assumed since all WASM functions
        //     currently return Uint8Array.

        if (event.data.stringify_out || event.data.jsonify_out) {
            d = await skyBtos(d).catch((e) => {
                output_conv_failure=`Failed to stringify output: ${e.message}`;
            })
        }

        if (event.data.jsonify_out) {
            try {
                d = JSON.parse(d)
            } catch(e) {
                output_conv_failure=`Failed to parse output to JSON: ${e.message}`;
            }
        }

        return d;
    });

    //=====================
    // TERMINATE THE WORKER
    //=====================
    // This will clear the current chunk from memory.

    //console.log(`Terminating worker ${event.data.round}`);
    postMessage({
        worker_id: event.data.worker_id,
        output: output,
        addtl: event.data.addtl,
        output_conv_failure: output_conv_failure,
        input_conv_failure: input_conv_failure,
    });
}