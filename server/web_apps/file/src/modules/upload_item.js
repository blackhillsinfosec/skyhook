
import React from "react";
import {fileApi} from "./file_api";
import {ListGroup,Badge} from "react-bootstrap";

export class UploadItem extends React.Component {
    constructor(props){
        super(props);
        this.cancel = this.cancel.bind(this);
    }

    async cancel(e){
        e.preventDefault();
        fileApi.cancelUpload(null, [this.props.webPath], null, this.props.getObfsConfig())
            .then((e) => {
                if(e.output.success){
                    this.props.reload();
                }else {
                    this.props.sendAlert(e.output.alert);
                }
            })
            .catch((e) => {
                this.props.sendAlert({
                    variant: 'danger',
                    heading: 'Failed to Cancel Upload',
                    message: `Cause: ${e.message}`
                });
            })
    }

    render(){
        return(
            <ListGroup.Item
                as={"li"}
                className={"d-flex justify-content-between align-items-start"}
            >
                <div className={"ms-2 me-auto"}>
                    <div className={"pb-2"}>
                        <div>{this.props.webPath}</div><Badge bg={"warning"} pill>Expiration: {this.props.expiration}</Badge> <Badge bg={"danger"} pill>
                        <a href={"#badge"} className={"link-light"} onClick={this.cancel}>
                            Cancel
                        </a>
                    </Badge>
                    </div>
                </div>
            </ListGroup.Item>
        );
    }
}


