import React from "react";
import {ListGroup} from "react-bootstrap";

export class DirItem extends React.Component {

    constructor(props){
        super(props);
        this.state = {

        };
    }

    render(){
        return (
            <ListGroup.Item
                as={"li"}
                className={"d-flex justify-content-between align-items-start"}
                variant={this.props.variant}
                onClick={(e) => {
                    e.preventDefault();
                    this.props.chDir(this.props.dirName);
                }}
            >
                <div className={"ms-2 me-auto"}>
                    <div className={"fw-bold"}><a href={"#dirChange"}>{this.props.dirName}</a></div>
                    Directory
                </div>
            </ListGroup.Item>
        );
    }
}

