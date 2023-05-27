/* global skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
/* exported skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */

import React from "react";
import {
    Row, Col, Form, ListGroupItem, InputGroup,
    Button, ProgressBar, Popover, PopoverBody,
    OverlayTrigger, ListGroup, FloatingLabel, ButtonGroup, ButtonToolbar, Tooltip
} from "react-bootstrap";
import {
    HouseFill,
    ArrowRepeat,
    ArrowLeftCircleFill,
    FilterCircleFill,
    FileEarmarkPlusFill,
    XCircleFill,
    ToggleOff,
    ToggleOn,
    EmojiWinkFill,
    EmojiSmileFill, Bezier2
} from "react-bootstrap-icons";
import {FileItem} from "./file_item";
import {DirItem} from "./dir_item";
import {UploadForm} from "./upload_form";
import {MD5, analyzeFile, regExpEscape} from "./misc_funcs";
import {ChunkFormFiles} from "./chunk_form_files";
import {fileApi} from "./file_api";

export class FileBrowser extends ChunkFormFiles {

    constructor(props){
        super(props);
        this.state = {
            upload_progress: null,
            filter_value: "",
            send_file_name: "",
            send_file_data: "",
            send_file_uploading: false,
            send_file_is_bin: false,
            show_obfs_config: false,
        }
        this.recvUploadProgress = this.recvUploadProgress.bind(this);
        this.updateField = this.updateField.bind(this);
        this.sendFile = this.sendFile.bind(this);
        this.chunksSent = this.chunksSent.bind(this);
        this.updateSendFileDataField = this.updateSendFileDataField.bind(this);
    }

    sendFile(e){
        e.preventDefault();
        this.setState({send_file_uploading:true});
        let fA = analyzeFile(this.state.send_file_data.length, this.props.maxChunkSize)
        this.chunkFormFiles(this.props.getCwd(), fA.file_chunk_size, fA.file_chunk_count, (prog) => {console.log(prog)})
    }

    registerTransfer(web_path, direction) {
        this.props.registerTransfer(web_path, direction);
    }

    deregisterTransfer(web_path) {
        this.props.deregisterTransfer(web_path);
    }

    getFiles() {
        let file = new Blob([this.state.send_file_data])
        file.name = this.state.send_file_name;
        return [file];
    }

    getObfConfig() {
        return this.props.getObfsConfig();
    }

    chunksSent(){
        this.props.sendAlert({
            variant: "success",
            message: "File upload complete",
            timeout: 3
        });
        this.setState({
            send_file_uploading: false,
            send_file_name: "",
            send_file_data: "",
            send_file_is_bin: false,
        });
        this.props.inspectDir(this.props.getCwd());
    }

    recvUploadProgress(p){
        this.setState({upload_progress:p})
    }

    updateSendFileDataField(e){
        e.preventDefault();
        if(e.type === "change"){
            // Just a field change, i.e., keyboard entry
            // Update field's text content
            this.updateField(e, "send_file_data");
        } else {
            //====================
            // HANDLE FILE PASTING
            //====================

            // Preserve pre-existing file name
            let fName = this.state.send_file_name;
            if(e.clipboardData.files.length && fName === "") {
                // Use pasted file's name
                fName = e.clipboardData.files[0].name;
            }

            // Append any text content to pre-existing text content
            for(let i=0; i<e.clipboardData.types.length; i++){
                let t = e.clipboardData.types[i]
                if(t.search("text") !== -1){
                    this.setState({
                        send_file_name: fName,
                        send_file_data: this.state.send_file_data + e.clipboardData.getData(t),
                        send_file_is_bin: false,
                    })
                    return
                }
            }

            // Preserve pre-existing content, otherwise the pasted content
            // becomes the field's value
            if(!this.state.send_file_data) {
                this.setState({
                    send_file_name: fName,
                    send_file_data: e.clipboardData.files[0],
                    send_file_is_bin: true,
                });
            }
        }
    }

    updateField(e, fName){
        e.preventDefault();
        this.setState({[fName]:e.target.value});
    }

    render(){

        //============================
        // CONSTRUCT DIRECTORY LISTING
        //============================

        let directories = [];
        let files = [];
        let dirEles;
        let fileEles;

        let dKeys = Object.keys(this.props.dirListing);

        // Split out file and directory names for sorting
        // Directories will be listed alphabetically at the top,
        // followed by files
        for (let i = 0; i < dKeys.length; i++) {
            // GET NAMES OF ALL FILES AND DIRECTORIES
            // These'll be sorted later
            let k = dKeys[i];
            let target = this.props.dirListing[k].is_dir ? directories : files;
            target.push(this.props.dirListing[k]);
        }


        let reg;
        if(this.state.filter_value !== ""){
            try {
                reg = new RegExp(this.state.filter_value);
            } catch(e) {
                reg = regExpEscape(this.state.filter_value);
            }
        }

        if (directories.length) {
            // Craft directory entries
            dirEles = [];
            for (let i = 0; i < directories.length; i++) {

                if(reg && directories[i].name.search(reg) === -1){ continue }

                let abs_path = `/${directories[i].name}`;
                if(this.props.getCwd() !== '/'){
                    abs_path = `${this.props.getCwd()}${abs_path}`;
                }

                dirEles.push(
                    <DirItem
                        key={`${directories[i].name}-dir-entry`}
                        dirName={directories[i].name}
                        variant={"dark"}
                        chDir={(e) => {
                            this.props.inspectDir(abs_path, true);
                        }}
                        absPath={abs_path}
                    />
                )
            }
        }

        if(dirEles && !dirEles.length){dirEles = undefined;}

        if(files.length){
            fileEles=[];
            for(let i=0; i<files.length; i++) {

                if(reg && files[i].name.search(reg) === -1){ continue }

                // Craft file entries
                let abs_path = `/${files[i].name}`;
                if(this.props.getCwd() !== '/'){
                    abs_path = `${this.props.getCwd()}${abs_path}`;
                }

                let fA = analyzeFile(files[i].size, this.props.maxChunkSize)

                fileEles.push(
                    <FileItem
                        key={`${files[i].name}-file-entry`}
                        storeName={MD5(abs_path)}
                        absPath={abs_path}
                        fileName={files[i].name}
                        fileSize={files[i].size}
                        humanBs={fA.human_bs}
                        status={"unstaged"}
                        maxChunkSize={this.props.maxChunkSize}
                        chunkSize={fA.file_chunk_size}
                        chunkCount={fA.file_chunk_count}
                        getObfsConfig={this.props.getObfsConfig}
                        registerTransfer={this.props.registerTransfer}
                        deregisterTransfer={this.props.deregisterTransfer}
                        stagingEnabled={this.props.stagingEnabled}
                        sendAlert={this.props.sendAlert}
                    />
                )
            }
        }

        if(fileEles && !fileEles.length){fileEles=undefined;}

        //======================
        // BUILD CURRENT LISTING
        //======================

        let lg_files;
        if(files.length || directories.length) {

            //====================================
            // COMPILE FILE AND DIRECTORY ELEMENTS
            //====================================

            lg_files = (
                <ListGroup as={"ol"}>
                    {dirEles && dirEles.length && dirEles}
                    {fileEles && fileEles.length && fileEles}
                </ListGroup>
            );

        }else {

            //========================================
            // PRESENT SOMETHING FOR EMPTY DIRECTORIES
            //========================================

            lg_files = (
                <ListGroup as={"ol"}>
                    <ListGroupItem as={"li"} className={"d-flex justify-content-between align-items-start"}>
                        <div className={"ms-2 me-auto"}>
                            <div className={"fw-bold"}>
                                Directory is empty
                            </div>
                        </div>
                    </ListGroupItem>
                </ListGroup>
            )
        }

        //===============
        // FILTER POPOVER
        //===============

        let filter_popover = (
            <Popover>
                <PopoverBody>
                    <InputGroup size={"sm"}>
                        <InputGroup.Text>Filter</InputGroup.Text>
                        <Form.Control
                            type={"text"}
                            value={this.state.filter_value}
                            onChange={(e) => {this.updateField(e, "filter_value")}}
                        />
                    </InputGroup>
                </PopoverBody>
            </Popover>
        )

        //==================
        // SEND FILE POPOVER
        //==================

        let send_file_popover = (
            <Popover>
                <PopoverBody>
                    <Form>
                    <Form.Group className={"mb-3"}>
                        <FloatingLabel label={this.props.dirListing[this.state.send_file_name] ? "File Name taken!" : this.state.send_file_name ? "File Name" : "File Name (Required)"}>
                        <Form.Control
                            type={"text"}
                            value={this.state.send_file_name}
                            onChange={(e) => {
                                this.updateField(e, "send_file_name")
                            }}
                        />
                        </FloatingLabel>
                    </Form.Group>
                    <Form.Group className={"mb-3"}>
                        <FloatingLabel label={this.state.send_file_data ? "File Content" : "File Content (Required)"}>
                        <Form.Control
                            as={"textarea"}
                            value={!this.state.send_file_is_bin ? this.state.send_file_data : "Binary content"}
                            disabled={this.state.send_file_is_bin}
                            onChange={this.updateSendFileDataField}
                            onPaste={this.updateSendFileDataField}
                        />
                        </FloatingLabel>
                    </Form.Group>
                    </Form>
                    <ButtonGroup size={"sm"} className={"w-100"}>
                        <Button
                            size={"sm"}
                            variant={"secondary"}
                            onClick={this.sendFile}
                            disabled={
                                this.props.dirListing[this.state.send_file_name] ||
                                !(this.state.send_file_name && this.state.send_file_data)
                            }
                        >
                            Send File
                        </Button>
                        <Button
                            size={"sm"}
                            variant={"warning"}
                            onClick={(e) => {
                                e.preventDefault();
                                this.setState({
                                    send_file_name: "",
                                    send_file_data: "",
                                    send_file_is_bin: false,
                                    send_file_uploading: false,
                                });
                            }}
                            disabled={
                                !(this.state.send_file_name || this.state.send_file_data)
                            }
                        >
                            Reset
                        </Button>
                    </ButtonGroup>
            </PopoverBody>
            </Popover>
        )

        let obf_input;

        if(this.state.show_obfs_config){
            obf_input =
            <Row>
                <Col>
                    <Form>
                        <FloatingLabel label={"Current Obfuscator Configuration"}>
                            <Form.Control
                                as={"textarea"}
                                disabled={true}
                                style={{height: '200px'}}
                                value={JSON.stringify(this.getObfConfig(), null, 4)}/>
                        </FloatingLabel>
                    </Form>
                </Col>
            </Row>
        }

        //========================
        // BROWSER BAR INPUT GROUP
        //========================

        let browser_bar = (
            <ButtonToolbar size={"sm"}>
                <ButtonGroup size={"sm"} className={"me-2"}>
                    <OverlayTrigger overlay={<Tooltip>Home</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            disabled={this.props.getCwd() === '/'}
                            onClick={(e) => {
                                e.preventDefault();
                                this.props.inspectDir('/', true);
                            }}
                        >
                            <HouseFill size={20}/>
                        </Button>
                    </OverlayTrigger>
                    <OverlayTrigger overlay={<Tooltip>Back</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            disabled={this.props.getCwd() === '/'}
                            onClick={(e) => {
                                e.preventDefault();
                                let s = this.props.getCwd().split('/');
                                s = s.slice(0,s.length-1).join('/')
                                this.props.inspectDir(s !== '' ? s : '/', true);
                            }}
                        >
                            <ArrowLeftCircleFill size={20}/>
                        </Button>
                    </OverlayTrigger>
                    <OverlayTrigger overlay={<Tooltip>Toggle Staging</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            onClick={( e) => {
                                e.preventDefault();
                                this.props.toggleStaging();
                            }}
                        >
                            {this.props.stagingEnabled ? <ToggleOn size={20}/> : <ToggleOff size={20}/>}
                        </Button>
                    </OverlayTrigger>
                    <OverlayTrigger overlay={<Tooltip>Reload File Listing</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            onClick={this.props.reload}
                        >
                            <ArrowRepeat size={23}/>
                        </Button>
                    </OverlayTrigger>
                </ButtonGroup>
                <ButtonGroup size={"sm"} className={"me-2"}>
                    <OverlayTrigger
                        trigger={"click"}
                        placement={"bottom"}
                        overlay={send_file_popover}>
                        <Button
                            variant={(this.state.send_file_name + this.state.send_file_data).length ? "warning" : "secondary"}
                            onClick={(e) => {e.preventDefault()}}
                        >
                            <FileEarmarkPlusFill size={20}/>
                        </Button>
                    </OverlayTrigger>
                    <OverlayTrigger
                        trigger={"click"}
                        placement={"right"}
                        overlay={filter_popover}>
                        <Button
                            variant={this.state.filter_value === '' ? "secondary" : "warning"}
                            onClick={(e) => {e.preventDefault()}}
                        >
                            <FilterCircleFill size={20}/>
                        </Button>
                    </OverlayTrigger>
                </ButtonGroup>
                <ButtonGroup size={"sm"} className={"me-2"}>
                    <OverlayTrigger overlay={<Tooltip>{this.state.show_obfs_config ? "Hide" : "Show"} Obfuscators</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            onClick={( e) => {
                                e.preventDefault();
                                this.setState({show_obfs_config:!this.state.show_obfs_config});
                            }}
                        >
                            {this.state.show_obfs_config ? <EmojiWinkFill size={20}/> : <EmojiSmileFill size={20}/>}
                        </Button>
                    </OverlayTrigger>
                    <OverlayTrigger overlay={<Tooltip>Reload Obfuscators</Tooltip>}>
                        <Button
                            variant={"secondary"}
                            onClick={( e) => {
                                e.preventDefault();
                                // this.props.inspectDir('/', true);
                                fileApi.getObfuscators().then(out => {
                                    if(out.output.obfs_config) {
                                        this.props.sendObfsConfig(out.output.obfs_config);
                                    }
                                    this.props.sendAlert(out.output.alert);
                                })
                            }}
                        >
                            <Bezier2 size={20}/>
                        </Button>
                    </OverlayTrigger>
                </ButtonGroup>
                <ButtonGroup size={"sm"} className={"me-2"}>
                    <OverlayTrigger overlay={<Tooltip>Log Out</Tooltip>}>
                        <Button
                            variant={"danger"}
                            onClick={( e) => {
                                e.preventDefault();
                                this.props.doLogout();
                            }}
                        >
                            <XCircleFill size={20}/>
                        </Button>
                    </OverlayTrigger>
                </ButtonGroup>
            </ButtonToolbar>
        );

        return (
            <Col className={"col-12 pb-1 border border-top-0 border-2 border-secondary border-opacity-25"}>
                <Row className={"mt-2 mb-2"}>
                    <Col>
                        <Row className={"mb-1"}>
                            <Col className={"mt-1"}>
                                <Form.Control disabled value={this.props.getCwd()} size={"sm"}/>
                            </Col>
                        </Row>
                        <Row className={"mb-1"}>
                            <Col className={"mt-1"}>
                                {browser_bar}
                            </Col>
                        </Row>
                        {obf_input}
                        <Row className={"justify-content-between"}>
                            <UploadForm
                                maxChunkSize={this.props.maxChunkSize}
                                getObfsConfig={this.props.getObfsConfig}
                                getCwd={this.props.getCwd}
                                registerTransfer={this.props.registerTransfer}
                                deregisterTransfer={this.props.deregisterTransfer}
                                sendUploadProgress={this.recvUploadProgress}
                                sendAlert={this.props.sendAlert}
                            />
                            {this.state.upload_progress != null && <Col>
                                <ProgressBar
                                    className={"mt-2 border border-secondary"}
                                    variant={"primary"}
                                    now={this.state.upload_progress}
                                    label={`${this.state.upload_progress - (this.state.upload_progress % 1)}%`}
                                    striped={true}
                                />
                            </Col>}
                            <Col className={"col-md-3 mt-1 mb-2"}>
                                <InputGroup size={"sm"}>
                                    <Form.Control
                                        type={"number"}
                                        value={this.props.maxChunkSize}
                                        onChange={this.props.sendMaxChunkSize}
                                        onBlur={this.props.sendMaxChunkSize}
                                    />
                                    <InputGroup.Text>Chunk Size (MB)</InputGroup.Text>
                                </InputGroup>
                            </Col>
                        </Row>
                        <Row>
                            <Col className={"mt-1"}>
                                {lg_files}
                            </Col>
                        </Row>
                    </Col>
                </Row>
            </Col>
        );
    }
}

