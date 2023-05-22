/* exported skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
/* global skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */

import React from "react";
import {addRangeHeader} from "./misc_funcs";
import {fileApi} from "./file_api";
import {MAX_WORKERS} from "./constants";
import {wait} from "./waiter";

// ChunkFormFiles provides methods and attributes to perform a chunked
// file upload to a Skyhook server.
export class ChunkFormFiles extends React.Component {

    constructor(props){
        super(props);

        if(!this.props.sendAlert){
            throw new Error('ChunkFormFiles children needs a sendAlert property to receive error alerts')
        }

        this.getFiles = this.getFiles.bind(this);
        this.uploadFinished = this.uploadFinished.bind(this);
        this.sendChunk = this.sendChunk.bind(this);
        this.getObfConfig = this.getObfConfig.bind(this);
        this.registerUpload = this.registerUpload.bind(this);
        this.registerTransfer = this.registerTransfer.bind(this);
        this.deregisterTransfer = this.deregisterTransfer.bind(this);
    }

    registerTransfer(webPath, direction){
        throw new Error("Child classes must implement registerTransfer");
    }

    deregisterTransfer(webPath){
        throw new Error("Child classes must implement deregisterTransfer");
    }

    async registerUpload(filePath){
        await fileApi.registerUpload({}, [filePath], null, this.getObfConfig(), true, "json")
            .then((e) => {
                if(!e.output.success){
                    this.props.sendAlert(e.output.alert);
                }
            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: "danger",
                    heading: "Upload Registration Failed",
                    message: `Does the upload OR file already exist? (Cause: ${e.message})`,
                });
                throw e;
            })
    }

    // getFiles is expected to return an array of files set
    // via web form.
    //
    // The return type should be an array.
    getFiles(){
        throw new Error("Component must implement a getFiles method")
    }

    // sendChunk is expected to send an individual chunk to a
    // Skyhook server.
    //
    // This method must ensure that the request is authenticated.
    async sendChunk(chunk, webPath, offset, rawSize) {
        // NOTE sendChunk signature: sendChunk(header, chunk)
        // - chunk is the fully encoded file chunk
        // - webPath is the encoded URL param indicating the upload
        // - offset indicates the numeric offset where the chunk
        //   should be inserted
        let headers = addRangeHeader(offset, offset+rawSize)

        let finished=false;
        let worker = new Worker(wasm_worker);
        worker.onmessage = (e) => {
            worker.terminate();
            chunk=e.data.output;
            finished=true;
        }
        worker.postMessage({
            wasm_exec: wasm_exec,
            algos_wasm: algos_wasm,
            wasm_helpers: wasm_helpers,
            func: "RunObfs",
            args: ["obf", chunk, this.getObfConfig()],
            bytefi_in: [1],
            stringify_out: true,
        })
        while(!finished){await wait(50)}

        await fileApi.putFileChunk(chunk, [webPath], headers, this.getObfConfig())
            .then((e) => {
                if(!e.output.success) {
                    this.props.sendAlert(e.output.alert);
                }
            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: 'danger',
                    heading: 'Failed to Send File Chunk',
                    message: `Cause: ${e.message}`
                })
                throw e;
            })

    }

    // getObfConfig is expected to return the current obfuscation
    // configuration. It's called by chunkFormFiles prior to processing
    // file chunks.
    getObfConfig() {
        throw new Error("Component must implement a getObfConfig method")
    }

    // chunksSent is called function once all chunks have been sent
    // to the server, allowing the component to modify its current
    // state.
    async uploadFinished(webPath) {
        await fileApi.uploadFinished({}, [webPath], null, this.getObfConfig())
            .then((e) => {
                if(!e.output.success) {
                    this.props.sendAlert(e.output.alert);
                }
            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: 'danger',
                    heading: 'Failed to Complete Upload',
                    message: `Cause: ${e.message}`
                })
            })
    }

    // chunkFormFiles method is responsible for uploading a file to the
    // server.
    //
    // It works by:
    //
    // - Reads the file in as an array buffer
    // - Creates a view into the bytes via Uint8Array
    // - Reads each chunk out of the Uint8Array
    // - Sends each chunk to the server for uploading
    async chunkFormFiles(cwd, chunk_size, chunk_count, prog_callback) {

        //=======================
        // PREPARE FOR PROCESSING
        //=======================

        let file = this.getFiles()[0];
        if (file === undefined) {
            throw new Error("No file found in array returned by getFiles method");
        }

        let reader = new FileReader();

        //====================
        // REGISTER THE UPLOAD
        //====================

        let webPath = "/" + file.name;
        if(cwd!=="/") {
            webPath = cwd + file.name;
        }

        await this.registerUpload(webPath)
            .catch((e) => {
                if(this.chunksSent){this.chunksSent()};
                throw e;
            })
        this.registerTransfer(webPath, "up");

        //=======================================
        // PROCESS CHUNKS ONCE THE FILE IS LOADED
        //=======================================

        let workers = {};
        let err_break;
        reader.onloadend = async (e) => {

            let data = new Uint8Array(reader.result);
            let bytes_sent = 0;

            for (let offset=0; offset < chunk_count; offset++) {

                if(err_break) { break }

                //============================================
                // WAIT UNTIL ONE OR MORE WORKERS ARE FINISHED
                //============================================

                while(Object.keys(workers).length >= MAX_WORKERS){
                    await wait(50);
                }

                //==================================
                // DERIVE OFFSETS AND GET FILE SLICE
                //==================================

                let start = offset * chunk_size;
                let end = start + chunk_size;
                let chunk = data.slice(start, end);

                workers[offset]=null;
                this.sendChunk(chunk, webPath, start, chunk.byteLength)
                    // eslint-disable-next-line
                    .then((e) => {
                        bytes_sent += chunk.byteLength
                        prog_callback(100 * (bytes_sent / data.byteLength))
                    })
                    // eslint-disable-next-line
                    .catch((e) => {
                        err_break = {
                            variant: "danger",
                            heading: "Failed to Upload File Chunk",
                            message: `Reason: ${e.message}`
                        }
                    })
                    .finally(() => {
                        delete workers[offset]
                    })
            }

            //====================================
            // WAIT UNTIL ALL WORKERS ARE FINISHED
            //====================================

            while (Object.keys(workers).length) {
                await wait(50)
            }

            if(!err_break) {
                prog_callback(null);

                // Notify component that the upload is complete
                await this.uploadFinished(webPath);

                // Run callback function upon completion.
                //
                // This allows components to apply state, etc.
                if (this.chunksSent) {
                    this.chunksSent();
                }
            } else {
                await fileApi.cancelUpload(null, webPath, null, this.getObfConfig())
                    .catch((e) => {
                        this.props.sendAlert({
                            variant: 'heading',
                            heading: 'Failed to Cancel Upload',
                            message: `Cause: ${e.message}`
                        })
                    });
                this.props.sendAlert(err_break);
            }
            this.deregisterTransfer(webPath);
        }

        // Read the file and send chunks to the server
        reader.readAsArrayBuffer(file);
    }
}
