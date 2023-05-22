import React from "react";
import Popover from 'react-bootstrap/Popover';
import OverlayTrigger from 'react-bootstrap/OverlayTrigger';
import {Row, Col, Button, ButtonGroup} from "react-bootstrap";
import {PencilFill, PersonXFill, Check, ClipboardFill, X, EyeFill, EyeSlashFill} from "react-bootstrap-icons";

export class UserButtonGroup extends React.Component {
    render(){

        let keyBase = `${this.props.username}-usr-btn-`
        let pop = (
            <Popover key={keyBase+'norm-del-user'}>
                <Popover.Header as={"h3"}>Delete {this.props.username} user?</Popover.Header>
                <Popover.Body>
                    <Row>
                        <Col>
                            <p>This cannot be undone!</p>
                        </Col>
                    </Row>
                    <Row className={"justify-content-center"}>
                        <Col>
                            <Button
                                className={"w-100"}
                                size={"sm"}
                                variant={"danger"}
                                onClick={this.props.deleteUser}
                            >
                                Confirm
                            </Button>
                        </Col>
                    </Row>
                </Popover.Body>
            </Popover>)

        let buttons = []
        if (this.props.mode === "normal") {
            buttons = [
                <Button
                    key={keyBase+"norm-show-cp-pass"}
                    variant={"secondary"}
                    onClick={this.props.showPassword}
                >
                    {
                        this.props.passwordShown &&
                        <EyeSlashFill size={18}/>
                    }{
                        !this.props.passwordShown &&
                        <EyeFill size={18}/>
                    }

                </Button>,
                <Button
                    key={keyBase+`norm-cp-pass`}
                    vairant={"primary"}
                    onClick={() => {
                        navigator.clipboard.writeText(`${this.props.username}:${this.props.password}:${this.props.token}`);
                        this.props.sendAlert({
                            variant:"success",
                            heading:"Credentials copied to clipboard",
                            message:this.props.username,
                            timeout:2});
                    }}
                >
                    <ClipboardFill size={18} data-toggle={"tooltip"} title={"Copy credentials"}/>
                </Button>,
                <Button key={keyBase+`norm-edit`} variant={"warning"}
                        onClick={() => {this.props.changeMode("edit")}}>
                    <PencilFill size={18}/>
                </Button>,
                <OverlayTrigger
                    key={keyBase+"-del-overlay"}
                    trigger={"focus"}
                    placement={"top"}
                    overlay={pop}
                >
                    <Button key={keyBase+`norm-del`} variant={"danger"}>
                        <PersonXFill size={20}/>
                    </Button>
                </OverlayTrigger>,
            ]
        } else if (this.props.mode === "edit") {
            buttons = [
                <Button
                    key={keyBase+`edit-save`}
                    variant={"warning"}
                    onClick={() => {
                        this.props.sendAlert({
                            variant: "success",
                            heading: "User Updated",
                            message: this.props.username,
                            timeout: 3});
                        this.props.changeMode("save-edit");
                    }}
                    disabled={!this.props.canSave}
                >
                    <Check size={23}/> Save
                </Button>,
                <Button
                    key={keyBase+`edit-cancel`}
                    variant={"secondary"}
                    onClick={() => {this.props.changeMode("cancel-edit")}}
                >
                    <X size={23} /> Cancel
                </Button>,
            ]
        } else if (this.props.mode === "new") {
            buttons = [
                <Button
                    key={keyBase+`edit-save-new`}
                    variant={"warning"}
                    onClick={() => {
                        this.props.sendAlert({
                            variant: "success",
                            heading: "New User Created",
                            message: this.props.username,
                            timeout: 3});
                        this.props.changeMode("save-edit-new");
                    }}
                    disabled={!this.props.canSave}
                >
                    <Check size={23}/> Save
                </Button>,
            ]
        }

        return(
            <ButtonGroup
                aria-label={"edit-user"}
                className={"btn-group-sm w-100"}
                children={buttons}/>
        )
    }
}
