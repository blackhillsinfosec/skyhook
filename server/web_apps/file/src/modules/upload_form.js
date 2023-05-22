/* global skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */
/* exported skyB64 skyStob skyBtos skyMd5Sum RunObfs algos_wasm wasm_exec wasm_helpers wasm_worker */

import React from "react";
import {Row,Col,OverlayTrigger,Button,
    Popover,PopoverHeader,PopoverBody,
    Spinner,InputGroup,Form} from "react-bootstrap";
import {InfoCircleFill} from "react-bootstrap-icons";
import {ChunkFormFiles} from "./chunk_form_files";
import {analyzeFile,numberWithCommas} from "./misc_funcs";

export class UploadForm extends ChunkFormFiles {

    constructor(props){
        super(props);
        this.uploadInput = React.createRef()
        this.state = {
            max_chunk_size: this.props.maxChunkSize,
            file_selected: false,
            uploading: false,
        };
        this.handleUpload = this.handleUpload.bind(this);
        this.reset = this.reset.bind(this);
        this.chunksSent = this.chunksSent.bind(this);
    }

    componentDidMount() {
        this.reset();
    }

    getFiles(){
        return this.uploadInput.current.files;
    }

    chunksSent(){
        this.file_chunk_count = null;
        this.setState({uploading: false});
        this.reset();
    }

    getObfConfig(){
        return this.props.getObfsConfig();
    }

    reset(){
        this.setState({
            timestamp: Date.now(),
            uploading: false,
            file_selected: false,
        });
    }

    registerTransfer(webPath, direction){
        this.props.registerTransfer(webPath, direction);
    }

    deregisterTransfer(webPath) {
        this.props.deregisterTransfer(webPath);
    }

    handleUpload(e){
        e.preventDefault();
        try {
            this.chunkFormFiles(this.props.getCwd(), this.file_chunk_size,
                this.file_chunk_count, this.props.sendUploadProgress)
            this.setState({uploading: true});
        } catch(err) {
            // TODO alert on failed upload initialization
            console.log(`Failed to initialize upload: ${err}`)
        }
    }

    render(){

        //========================
        // ANALYZE ANY TARGET FILE
        //========================

        let human_bs = "";
        if(this.state.file) {
            let fA = analyzeFile(this.state.file.size, this.props.maxChunkSize);
            this.file_chunk_size = fA.file_chunk_size;
            this.file_chunk_count = fA.file_chunk_count;
            human_bs = fA.human_bs;
        }

        //=============
        // INFO POPOVER
        //=============

        let info;
        if(this.state.file_selected){
            let popover = (
                <Popover>
                    <PopoverHeader as={"h3"}>Upload Information</PopoverHeader>
                    <PopoverBody>
                        <Row>
                            <Col>Size</Col>
                            <Col>{numberWithCommas(human_bs)}</Col>
                        </Row>
                        <Row>
                            <Col className={"text-nowrap"}>Chunk Count</Col>
                            <Col>{numberWithCommas(this.file_chunk_count)} x {this.props.maxChunkSize}MB</Col>
                        </Row>
                    </PopoverBody>
                </Popover>
            );
            info = (
                <OverlayTrigger
                    key={"browser-upload-overlay"}
                    trigger={["hover","focus"]}
                    placement={"bottom"}
                    overlay={popover}>
                    <Button
                        size={"sm"}
                        variant={"secondary"}
                    >
                        <InfoCircleFill size={15}/>
                    </Button>
                </OverlayTrigger>
            );
        }

        //===================
        // UPLOAD BUTTON TEXT
        //===================

        let uptext = "Upload";
        if(this.state.uploading){
            uptext = <Spinner
                key={"browser-upload-spinner"}
                as={"span"}
                animation={"border"}
                size={"sm"}
                role={"status"}
            />
        }

        //======================
        // UPLOAD & INFO BUTTONS
        //======================

        let buttons;
        if(this.state.file_selected) {
            buttons = [
                <Button
                    key={"browser-upload-button"}
                    variant={"secondary"}
                    onClick={this.handleUpload}
                    size={"sm"}
                    disabled={this.state.uploading}
                >
                    {uptext}
                </Button>,
                info
            ];
        }

        return (
            <Col key={this.state.timestamp} className={"mt-1 col-md-5"}>
                <Form encType={"multipart/form-data"} method={"post"}>
                    <Row>
                        <Form.Group as={Col}>
                            <InputGroup size={"sm"}>
                                <Form.Control
                                    type={"file"}
                                    size={"sm"}
                                    ref={this.uploadInput}
                                    disabled={this.state.uploading}
                                    onChange={(e) => {
                                        if(e.target.value) {
                                            this.setState({
                                                file: this.uploadInput.current.files[0],
                                                file_selected: true,
                                            })
                                        }
                                    }}
                                />
                                {buttons}
                            </InputGroup>
                        </Form.Group>
                    </Row>
                </Form>
            </Col>
        );
    }
}
