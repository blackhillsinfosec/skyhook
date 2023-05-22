onmessage = async (event) => {

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

    //==================
    // RUN WASM FUNCTION
    //==================

    // Load wasm_helpers
    eval(event.data.wasm_helpers);

    // Run the target function
        // Passing the function name to eval effectively
        // returns a reference to the function, which is
        // then called directly.
    let output = await eval(event.data.func)(...event.data.args).then(async (d) => {
        if (event.data.stringify) {
            return await skyBtos(d)
        }
    });

    //=====================
    // TERMINATE THE WORKER
    //=====================
    // This will clear the current chunk from memory.

    //console.log(`Terminating worker ${event.data.round}`);
    postMessage({
        worker_id: event.data.worker_id,
        output: output,
    });
}