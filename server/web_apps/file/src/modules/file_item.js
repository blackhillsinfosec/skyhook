/* global skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
/* exported skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */

import {Badge, ListGroup, ProgressBar} from "react-bootstrap";
import {NAMES_DB_NAME, CHUNKS_STORE_NAME, MAX_WORKERS, MAX_STAGING_REQ} from "./constants";
import React from "react";
import {wait} from "./waiter";
import {fileApi} from "./file_api";
import {addRangeHeader} from "./misc_funcs";

var MD5 = require('md5');

export class FileItem extends React.Component {

    constructor(props){
        super(props);
        this.state={
            status: this.props.status,
        };
        this.stageFile = this.stageFile.bind(this);
        this.unstageFile = this.unstageFile.bind(this);
        this.saveStagedFile = this.saveStagedFile.bind(this);
        this.downloadPurge = this.downloadPurge.bind(this);
    }

    componentDidMount() {

        //=========================
        // DETERMINE STATUS OF FILE
        //=========================

        let md5 = MD5(this.props.absPath)

        // We assume the file is staged if the object store appears
        // in IndexedDB and the file is neither staging or saving.
        let req = window.indexedDB.open(NAMES_DB_NAME, 1);
        req.onsuccess = (e) => {
            let t = req.result.transaction(NAMES_DB_NAME, "readonly");
            let obj = t.objectStore(NAMES_DB_NAME).get(md5)
            obj.onsuccess = (e) => {
                if(obj.result){
                    if(obj.result.staged){
                        this.setState({status: "staged"});
                    }else{
                        this.setState({status: "staging"});
                    }
                }
            }
        }
    }

    async downloadPurge(e){
        await this.stageFile(e);
        while(this.state.status !== "staged"){
            await wait(50);
        }
        await this.saveStagedFile(e, "unstaged");
        await this.unstageFile(e);
    }

    // TODO finish this method
    // TODO See chunkUpload for donor logic to be used for streaming
    //    chunks to disk
    // Download and stage a file in IndexedDB.
    async stageFile(e){
        e.preventDefault();

        //======
        // NOTES
        //======
        // We perform downloads at the FileBrowser to ensure that transfers
        // are retained when the user changes directories.
        //======

        let obfs_config = this.props.getObfsConfig();
        this.props.registerTransfer(this.props.absPath, "down");

        // Database name is the MD5 of the absolute file path.
        let req = window.indexedDB.open(this.props.storeName, 1);

        req.onupgradeneeded = (e) => {
            req.result.createObjectStore(CHUNKS_STORE_NAME, {keyPath: "offset"})

            let nreq = window.indexedDB.open(NAMES_DB_NAME, 1);
            nreq.onsuccess = (e) => {
                let t = nreq.result.transaction(NAMES_DB_NAME, "readwrite");

                // We store the obfs_config alongside the name to ensure that
                // staged files can be deobfuscated even after the obfuscation
                // config is changed.
                t.objectStore(NAMES_DB_NAME).put({
                    name:this.props.storeName,
                    staged: false,
                    obfs_config: obfs_config,
                })
            }
        }

        let awaiting = {};
        let err_break=false;
        req.onsuccess = async (event) => {

            this.setState({status: "staging"});
            let db=req.result;

            //=============================
            // REQUEST CHUNKS IN WEB WORKER
            //=============================

            for (let offset = 0; offset < this.props.chunkCount; offset++) {

                if(err_break){break}

                while(Object.keys(awaiting).length >= MAX_STAGING_REQ){
                    await wait(50);
                }
                awaiting[offset]=null;

                //===================
                // CRAFT RANGE HEADER
                //===================

                let start = offset * this.props.chunkSize;
                let end = start + this.props.chunkSize-1;
                if(offset === this.props.chunkCount-1 || end > this.props.fileSize){
                    end = this.props.fileSize
                }
                let headers = addRangeHeader(start,end)

                //==================
                // REQUEST THE CHUNK
                //==================

                fileApi.downloadFileChunk({}, [this.props.absPath], headers, obfs_config, false)
                    // eslint-disable-next-line
                    .then((e) => {
                        let store = db.transaction(CHUNKS_STORE_NAME, "readwrite").objectStore(CHUNKS_STORE_NAME);
                        if(!e.output.success){
                            err_break = e.output.alert;
                        } else {
                            store.add({offset: offset, chunk: e.output.chunk});
                            this.setState({staged_progress: 100 * (end / this.props.fileSize)});
                        }
                    })
                    // eslint-disable-next-line
                    .catch((err) => {
                        err_break = {
                            variant: "danger",
                            heading: "Staging Error",
                            message: `Retry download/staging (${err.message})`,
                            timeout: 5,
                        }
                    }).finally(() => {
                        delete awaiting[offset];
                    })
            }

            // Allow all requests to complete
            while(Object.keys(awaiting).length){await wait(50);}

            let nreq = window.indexedDB.open(NAMES_DB_NAME, 1);
            nreq.onsuccess = (e) => {
                let store = nreq.result.transaction(NAMES_DB_NAME, "readwrite").objectStore(NAMES_DB_NAME);
                if(!err_break) {
                    // UPDATE TO STAGED
                    store.put({name: this.props.storeName, staged: true, obfs_config: obfs_config})
                    this.setState({status: "staged", staged_progress: 0});
                } else {
                    // REMOVE FAILED DOWNLOAD
                    store.delete(this.props.storeName);
                    this.setState({status: "unstaged", staged_progress: 0});
                    this.props.sendAlert(err_break);
                }
                this.props.deregisterTransfer(this.props.absPath);
            }
        }
    }

    // Delete the object store from indexed db.
    unstageFile(e){
        e.preventDefault();
        window.indexedDB.deleteDatabase(this.props.storeName);
        let nreq = window.indexedDB.open(NAMES_DB_NAME, 1);
        nreq.onsuccess = (e) => {
            let t = nreq.result.transaction(NAMES_DB_NAME, "readwrite");
            t.objectStore(NAMES_DB_NAME).delete(this.props.storeName);
            this.setState({status: "unstaged"});
        }
    }

    // Reassemble a file staged in IndexedDB and save it to disk.
    async saveStagedFile(e, after_status) {
        e.preventDefault();

        let obfs_config = this.props.getObfsConfig();

        //======================================
        // OPEN THE DATABASE FOR THE STAGED FILE
        //======================================

        let req = window.indexedDB.open(this.props.storeName);

        req.onerror = (errE) => {
            throw new Error(`IndexedDB database for file transfer not found: ${this.props.storeName}`);
        }

        req.onupgradeneeded = (upgE) => {
            throw new Error(`IndexedDB database for file transfer not found: ${this.props.storeName}`);
        }

        //===========================================
        // SUCCESSFUL CONNECTION TO DATABASE OCCURRED
        //===========================================

        let key_count;
        req.onsuccess = async (succE) => {

            //=======================================
            // PREPARE FOR THE INDEXEDDB TRANSACTIONS
            //=======================================

            let db = req.result;

            let trans = db.transaction(CHUNKS_STORE_NAME, "readonly");
            let store = trans.objectStore(CHUNKS_STORE_NAME);
            this.setState({status: "saving"});

            //========================
            // GET INDEXEDDB KEY COUNT
            //========================

            let count_req = store.count()

            count_req.onsuccess = (e) => {
                key_count = e.target.result;
            }

            count_req.onerror = (e) => {
                key_count = -1;
            }

        }

        while(!key_count){
            await wait(50);
        }

        if(key_count === -1){
            throw new Error("Failed to get indexed DB key count for download")
        }

        //==================================
        // ITERATE OVER KEYS (CHUNK OFFSETS)
        //==================================

        // Tracks workers in progress
          // offset -> chunk
        let workers={};

        let content = new Uint8Array(this.props.fileSize);
        let bytes_written = 0;
        let err_break=false;

        for(let offset=0; offset<key_count; offset++){

            if(err_break){ break }

            // Do not exceed MAX_WORKERS
            while(Object.keys(workers).length >= MAX_WORKERS){
                await wait(50);
            }

            // Track the worker by offset
            workers[offset]=null;

            // Open a fresh connection to the target database
            // Note: We create a fresh connection for each chunk because
            //   IndexedDB disallows async in events, which results in
            //   the DB connection being severed after each call to await.
            //   This works around the issue by providing a connection for
            //   each chunk.
            let req = window.indexedDB.open(this.props.storeName);
            // eslint-disable-next-line
            req.onsuccess = async (succE) => {

                //===============================
                // RETRIEVE AND DEOBFUSCATE CHUNK
                //===============================

                let db = req.result;
                let trans = db.transaction(CHUNKS_STORE_NAME, "readonly");
                let store = trans.objectStore(CHUNKS_STORE_NAME);
                let get_req = store.get(offset);

                get_req.onsuccess = async (e) => {

                    // Start a new worker to process the chunk
                    let worker = new Worker(wasm_worker);

                    // Response from worker will yield the decoded data, which
                    // will be suffixed to the end of the final content
                    worker.onmessage = async (wE) => {
                        worker.terminate();

                        //======================================
                        // WAIT FOR EARLIER CHUNKS TO BE WRITTEN
                        //======================================

                        let ready=false;
                        while(!ready){
                            let keys = Object.keys(workers);
                            if(keys.length === 1){
                                // Assume that the only remaining key is the current worker's.
                                ready=true;
                            } else {
                                for(let i=0; i<keys.length && !ready; i++){
                                    if(keys[i] < wE.data.worker_id){
                                        // Wait for previous chunks to be written to content variable.
                                        await wait(50);
                                        break
                                    }else if(i+1 === keys.length){
                                        ready=true;
                                    }
                                }
                            }
                        }

                        //========================
                        // WRITE THE CURRENT CHUNK
                        //========================

                        content.set(wE.data.output, bytes_written);
                        bytes_written += wE.data.output.byteLength;
                        delete workers[wE.data.worker_id];
                        this.setState({staged_progress: 100 * (bytes_written / this.props.fileSize)});
                    }

                    worker.onerror = (e) => {
                        err_break=true;
                        delete workers[offset];
                        this.props.sendAlert({
                            variant: "danger",
                            message: "Failed to save file",
                            timeout: 10,
                            show: true,
                        });
                    }

                    //=================
                    // START THE WORKER
                    //=================

                    worker.postMessage({
                        wasm_exec: wasm_exec,
                        algos_wasm: algos_wasm,
                        wasm_helpers: wasm_helpers,
                        func: "RunObfs",
                        args: ["deobf", e.target.result.chunk, obfs_config],
                        worker_id: e.target.result.offset,
                        bytefi_in: [1],
                        stringify_out: false,
                        addtl: e.target.result.offset
                    })
                }
            }
        }

        // Wait for all chunks to be written to content
        while (Object.keys(workers).length) {await wait(50);}

        if(!err_break) {

            //========================
            // ENSURE FILE SIZES MATCH
            //========================

            // TODO finish this
            if (this.props.fileSize !== content.byteLength) {
                // TODO alert on this error and potentially force re-download
                console.log(`Deobfuscated to incorrect file size. Expected ${this.props.fileSize} but got ${content.byteLength}`)
            }

            //==================
            // DOWNLOAD THE FILE
            //==================

            content = new Blob([content], {type: 'octet/stream'});
            let a = document.createElement('a');
            let u = window.URL.createObjectURL(content);
            document.body.appendChild(a)
            a.style = 'display:none';
            a.href = u;
            a.download = this.props.fileName;
            a.click()
            window.URL.revokeObjectURL(u);

            let state = Object.assign({}, this.state);
            state.status = after_status ? after_status : "staged"
            state.staged_progress = 0;
            this.setState(state);
        }
    }

    render(){

        // TODO update all href links to actually perform an action
        let badges;
        let staged_badge;

        if(this.props.fileSize === 0 ){
            badges = [<Badge bg={"danger"} key={`${MD5(this.props.fileName)}-dd-empty-badge`}  className={"ms-2"} pill>Empty File (Unsupported)</Badge>];
        }else if(this.state.status === "staged"){
            staged_badge = <Badge bg={"success"} className={"ms-2"} pill>Staged</Badge>;
            badges = [
                <Badge
                    key={`${MD5(this.props.fileName)}-dd-save-to-disk-badge`}
                    bg={"warning"} className={"ms-1 me-1"}
                    pill
                >
                    <a
                        href={"#badge"}
                        className={"link-light"}
                        onClick={this.saveStagedFile}
                    >
                        Save to Disk
                    </a>
                </Badge>,
                <Badge
                    key={`${MD5(this.props.fileName)}-dd-delete-staged-badge`}
                    bg={"danger"}
                    pill
                >
                    <a
                        href={"#badge"}
                        className={"link-light"}
                        onClick={this.unstageFile}
                    >
                        Purge
                    </a>
                </Badge>
            ];
        } else if (this.state.status === "unstaged") {
            if(this.props.stagingEnabled) {
                badges = [
                    <Badge
                        key={`${MD5(this.props.fileName)}-dd-staged-badge}`}
                        bg={"secondary"}
                        className={"ms-1 me-1"}
                        pill
                    >
                        <a
                            href={"#badge"}
                            onClick={this.downloadPurge}
                            className={"link-light"}>Download & Purge</a>
                    </Badge>,
                    <Badge
                        key={`${MD5(this.props.fileName)}-dd-stage-badge}`}
                        bg={"primary"}
                        className={"me-1"}
                        pill>
                        <a
                            href={"#badge"}
                            className={"link-light"}
                            onClick={this.stageFile}
                        >
                            Stage
                        </a>
                    </Badge>
                ]
            } else {
                badges = [
                    <Badge
                        key={`${MD5(this.props.fileName)}-dd-staged-badge}`}
                        bg={"secondary"}
                        className={"ms-1 me-1"}
                        pill
                    >
                        <a
                            href={"#badge"}
                            onClick={this.downloadPurge}
                            className={"link-light"}>Download</a>
                    </Badge>,
                ]
            }
        } else if (this.state.status === "staging" || this.state.status === "saving") {
            badges = [
                <Badge
                    key={`${MD5(this.props.fileName)}-dd-staging-badge}`}
                    bg={this.state.status === "staging" ? "success" : "warning"}
                    className={"ms-1 me-1"}
                    pill
                >
                    {this.state.status.slice(0,1).toUpperCase()+this.state.status.slice(1)}
                </Badge>
            ]

            let prog = this.state.staged_progress === undefined ? 0 : this.state.staged_progress
            staged_badge = (
                <div className={"ms-5 mt-auto mb-auto w-25"}>
                <ProgressBar key={String(Date.now())} className={"border border-secondary"} striped variant={"primary"} label={`${prog - (prog % 1)}%`} now={prog}/>
                </div>
            );
        }

        return(
            <ListGroup.Item
                as={"li"}
                className={"d-flex justify-content-between align-items-start"}
            >
                <div className={"ms-2 me-auto"}>
                    <div className={"fw-bold"}>
                        {this.props.fileName} {badges}
                    </div>
                    {this.props.humanBs}
                </div>
                {staged_badge && staged_badge}
            </ListGroup.Item>
        );
    }
}

