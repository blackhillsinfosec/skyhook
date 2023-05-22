import React from "react";
import {Row, Col, ListGroup, ListGroupItem, Button, InputGroup} from "react-bootstrap";
import {ArrowRepeat} from "react-bootstrap-icons";
import {UploadItem} from "./upload_item";
import {fileApi} from "./file_api";

export class UploadBrowser extends React.Component {

    constructor(props){
        super(props);
        this.state={uploads: []};
        this.reload = this.reload.bind(this);
    }

    componentDidMount() {
        this.reload();
    }

    reload(){
        fileApi.listUploads(null, null, null, this.props.getObfsConfig())
            .then((e) => {
                if(e.output.success) {
                    this.setState({uploads: e.output.uploads ? e.output.uploads : []});
                }else{
                   this.props.sendAlert(e.output.alert);
                }
            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: 'danger',
                    heading: `Failed to List Uploads`,
                    message: `Cause: ${e.message}`,
                })
            });
    }

    render(){

        let lg_uploads;
        if(this.state !== undefined && this.state.uploads.length) {
            let uploads = [];
            for(let i=0; i<this.state.uploads.length; i++){
                let upload = this.state.uploads[i];
                uploads.push(
                    <UploadItem
                        key={upload.rel_path}
                        webPath={upload.rel_path}
                        absPath={upload.abs_path}
                        expiration={upload.expiration}
                        reload={this.reload}
                        getObfsConfig={this.props.getObfsConfig}
                        sendAlert={this.props.sendAlert}
                    />
                );
            }
            lg_uploads = <ListGroup as={"ol"}>{uploads}</ListGroup>;
        } else {
            lg_uploads = (
                <ListGroup as={"ol"}>
                    <ListGroupItem as={"li"} className={"d-flex justify-content-between align-items-start"}>
                        <div className={"ms-2 me-auto"}>
                            <div className={"fw-bold"}>No uploads registered</div>
                        </div>
                    </ListGroupItem>
                </ListGroup>
            );
        }

        return (
            <Col className={"col-12 pb-1 border border-top-0 border-2 border-secondary border-opacity-25"}>
                <Row className={"mt-2 mb-2"}>
                    <Col>
                        <Row className={"mb-1"}>
                            <Col className={"mt-1"}>
                                <InputGroup size={"sm"}>
                                    <Button
                                        variant={"secondary"}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            this.reload();
                                        }}
                                    >
                                        <ArrowRepeat size={23}/>
                                    </Button>
                                </InputGroup>
                            </Col>
                        </Row>
                        <Row>
                            <Col className={"mt-1"}>{lg_uploads}</Col>
                        </Row>
                    </Col>
                </Row>
            </Col>
        );
    }
}


