/* global skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
/* exported skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
import React from "react";
import {Nav, Row} from "react-bootstrap";
import {ArrowUp, FolderFill} from "react-bootstrap-icons";
import {fileApi} from "./file_api";
import {NAMES_DB_NAME, MEGABYTE, REC_CHUNK_SIZE, MAX_CHUNK_SIZE, STAGING_ENABLED_NAME} from "./constants";
import {mbsToBs} from "./misc_funcs";
import {FileBrowser} from "./file_browser";
import {UploadBrowser} from "./upload_browser";

export class BrowserPane extends React.Component {

    constructor(props){
        super(props);

        this.state = {
            mode: "browser",

            staging_enabled: false,

            // Default maximum chunk size.
            max_chunk_size: REC_CHUNK_SIZE,
            file_chunk_size: null,

            obfs_config:  this.props.recvObfsConfig(),

            // Will be an array of objects that describe ongoing
            // file transfers.
            transfers: {},

            // Determines if the obfuscator config form is disabled,
            // allowing us to disable other aspects of the interface
            // when it is being disabled.
            //
            // This prevents file transfers from getting borked due to
            // changes in the obfuscator.
            obfs_config_form_disabled: false,

            // Current working directory.
            cwd: "/",
            cwd_listing: {},
        };

        this.updateMaxChunkSize = this.updateMaxChunkSize.bind(this);
        this.transferCount = this.transferCount.bind(this);
        this.recvObfsConfigFormUpdate = this.recvObfsConfigFormUpdate.bind(this);
        this.recvObfsConfigFormDisabled = this.recvObfsConfigFormDisabled.bind(this);
        this.registerTransfer = this.registerTransfer.bind(this);
        this.deregisterTransfer = this.deregisterTransfer.bind(this);
        this.toggleStaging = this.toggleStaging.bind(this);
        this.logout = this.logout.bind(this);

        this.reloadFiles = this.reloadFiles.bind(this);
        this.inspectDir = this.inspectDir.bind(this);
        this.sendCwd = this.sendCwd.bind(this);
        this.sendObfsConfig = this.sendObfsConfig.bind(this);
    }

    componentDidMount(){

        let staging_enabled = localStorage.getItem(STAGING_ENABLED_NAME);
        if(!staging_enabled){
            localStorage.setItem(STAGING_ENABLED_NAME, "false")
        }

        //========================================
        // INITIALIZE STATE & DIRECTORY INSPECTION
        //========================================

        let state = {
            file_chunk_size: mbsToBs(this.state.max_chunk_size),
            staging_enabled: staging_enabled === "true"
        }

        if(!this.state.obfs_config){
            state.mode = "config"
            state.alert = {
                heading: "Obfuscator Configuration Required",
                message: "Obtain this from the admin panel.",
                variant: "danger",
                timeout: 7
            };
        }

        if(this.props.authenticated && this.state.obfs_config) {this.inspectDir("/");}
        this.setState(state);

        //=====================
        // INITIALIZE INDEXEDDB
        //=====================

        let req = window.indexedDB.open(NAMES_DB_NAME, 1)
        req.onupgradeneeded = (e) => {
            req.result.createObjectStore(NAMES_DB_NAME, {keyPath: "name"})
        }

    }

    async logout(){
        let out = fileApi.postLogout();
        if(out.output.alert){
            this.props.sendAlert(out.alert);
        }
    }

    toggleStaging(){
        localStorage.setItem(STAGING_ENABLED_NAME, String(!this.state.staging_enabled));
        this.setState({staging_enabled:!this.state.staging_enabled});
    }

    sendObfsConfig(){
        return this.state.obfs_config;
    }

    // Send the current working directory to the caller.
    sendCwd(){
        return this.state.cwd;
    }

    // Reload files without changing the current directory.
    reloadFiles(e){
        e.preventDefault();
        this.inspectDir(this.state.cwd);
    }

    // Inspect files for the target directory, which should be an absolute
    // server-aware path.
    async inspectDir(target, chDir){

        fileApi.inspectFiles(null, [target], null, this.state.obfs_config)
            .then((e) => {
                if(e.output.success) {

                    //=======================
                    // UPDATE COMPONENT STATE
                    //=======================

                    let entries = {};

                    if (e.output.listing && e.output.listing.length) {
                        for (let i = 0; i < e.output.listing.length; i++) {
                            entries[e.output.listing[i].name] = e.output.listing[i]
                        }
                    }

                    this.setState({
                        cwd_listing: entries,
                        cwd: chDir ? target : this.state.cwd
                    })

                } else {

                    this.props.sendAlert(e.output.alert);

                }

            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: 'danger',
                    heading: `Failed to List Files (${e.message})`,
                    message: 'Is the obfuscation config properly applied?'
                })
            })

    }

    transferCount(){
        return Object.keys(this.state.transfers).length;
    }

    registerTransfer(webPath, direction){
        if(this.state.transfers[webPath]){ throw new Error("Transfer already exists."); }
        if(direction !== "up" && direction !== "down"){
            throw new Error("direction must be either 'up' or 'down'");
        }
        let tr = Object.assign({}, this.state.transfers);
        tr[webPath] = direction
        this.setState({transfers: tr})
    }

    deregisterTransfer(webPath){
        if(!this.state.transfers[webPath]){ throw new Error("Unknown transfer specified."); }
        let tr = Object.assign({}, this.state.transfers);
        delete tr[webPath]
        this.setState({transfers: tr})
    }

    recvObfsConfigFormDisabled(disabled){
        this.setState({obfs_config_form_disabled: disabled})
    }

    recvObfsConfigFormUpdate(obfs_config){
        this.props.sendObfsConfig(obfs_config);
        this.setState({obfs_config: obfs_config});
    }

    updateMaxChunkSize(event){
        let value=event.target.value;
        if(event.type === "blur") {
            if (event.target.value > MAX_CHUNK_SIZE || event.target.value <= 0) {
                // Set to minimum chunk size.
                value = REC_CHUNK_SIZE;
                // TODO send alert regarding maximum chunk size
            }
            value=Number(value);
            if(isNaN(value)){
                value=0;
            }
        }

        this.setState({
            max_chunk_size: value,
            max_chunk_bytes: value*MEGABYTE
        });
    }

    render(){

        //========
        // NAV BAR
        //========

        let nav = (
            <Nav fill variant={"tabs"}>
                <Nav.Item>
                    <Nav.Link
                        href={"#browser"}
                        active={this.state.mode === "browser"}
                        onClick={(e) => {
                            e.preventDefault();
                            this.setState({mode:"browser"});
                        }}
                    >
                        <FolderFill size={20}/> Browser
                    </Nav.Link>
                </Nav.Item>
                <Nav.Item>
                    <Nav.Link
                        href={"#uploads"}
                        eventKey={"uploads"}
                        active={this.state.mode === "uploads"}
                        onClick={(e) => {
                            e.preventDefault();
                            this.setState({mode:"uploads"})
                        }}
                    >
                        <ArrowUp size={20}/>Uploads
                    </Nav.Link>
                </Nav.Item>
            </Nav>
        );

        //================
        // FINAL COMPONENT
        //================

        return (
            <Row className={"mt-2 mb-2"}>
                <Row>
                    {nav}
                </Row>
                <Row className={this.state.mode === "browser" ? "" : "d-none"}>
                    <FileBrowser
                        obfConfig={this.state.obfs_config}
                        maxChunkSize={this.state.max_chunk_size}
                        sendMaxChunkSize={this.updateMaxChunkSize}
                        dirListing={this.state.cwd_listing}
                        reload={this.reloadFiles}
                        inspectDir={this.inspectDir}
                        getCwd={this.sendCwd}
                        getObfsConfig={this.sendObfsConfig}
                        registerTransfer={this.registerTransfer}
                        deregisterTransfer={this.deregisterTransfer}
                        stagingEnabled={this.state.staging_enabled}
                        sendAlert={this.props.sendAlert}
                        sendObfsConfig={this.recvObfsConfigFormUpdate}
                        toggleStaging={this.toggleStaging}
                        doLogout={this.logout}
                    />
                </Row>
                <Row className={this.state.mode === "uploads" ? "" : "d-none"}>
                    <UploadBrowser
                        getObfsConfig={this.sendObfsConfig}
                        sendAlert={this.props.sendAlert}
                    />
                </Row>
            </Row>
        )
    }
}