import React from "react";
import {Form, InputGroup, Button, Card} from "react-bootstrap";
import { UserButtonGroup } from "./user_button_group";
import {ClipboardFill} from "react-bootstrap-icons";

export class UserForm extends React.Component {
    constructor(props){
        super(props);
        this.state = {
            user: props.user,
            mode: this.props.mode === "new" ? "new" : "normal",
            deleted: false,
            username: props.username,
            password: props.password,
            is_admin: props.is_admin,
            token: props.token,
            show_password: false,
        };
        this.changeMode = this.changeMode.bind(this);
        this.fieldChange = this.fieldChange.bind(this);
        this.delete = this.delete.bind(this);
        this.toStateObj = this.toStateObj.bind(this);
        this.toPropObj = this.toPropObj.bind(this);
        this.toggleShowPassword = this.toggleShowPassword.bind(this);
        this.fieldsPopulated = this.fieldsPopulated.bind(this);
    }

    toggleShowPassword(){
        this.setState({show_password:!this.state.show_password});
    }

    // Delete the current user.
    delete(){
        this.props.deleteUser(this.state.username);
    }

    // Return current user state.
    toStateObj(){
        return {
            username: this.state.username,
            password: this.state.password,
            is_admin: this.state.is_admin,
            token: this.state.token,
        }
    }

    // Return ORIGINAL user state, as described by properties.
    toPropObj(){
        return {
            username: this.props.username,
            password: this.props.password,
            is_admin: this.props.is_admin,
            token: this.props.token,
        }
    }

    // Change the current mode of the user, i.e. the form.
    changeMode(mode) {
        let state
        if (mode === "edit") {

            //====================
            // CHANGE TO EDIT MODE
            //====================

            state = {mode: "edit"};

        } else if (mode === "cancel-edit") {

            //===============================
            // RESTORE ORIGINAL USER SETTINGS
            //===============================

            state = this.toPropObj()
            state.mode = "normal"

        } else if (mode === "save-edit") {

            //=============
            // SAVE CHANGES
            //=============

            if(this.props.updateUser(this.props.username, this.toStateObj())) {
                state = {mode: "normal"};
            } else {
                return
            }

        } else if (mode === "save-edit-new"){

            //==============
            // SAVE NEW USER
            //==============

            this.props.addUser(this.toStateObj());
            state = {
                mode: "new",
                username:"",
                password:"",
                is_admin:false
            };

        }

        this.setState(state)
    }

    // Update the form field with a new value.
    fieldChange(name, event){
        this.setState({
            [name]: event.target.value
        })
    }

    fieldsPopulated(){
        return this.state.username && this.state.password && this.state.token;
    }

    render(){

        let canSave = this.state.mode === "new";
        if (this.state.mode === "edit") {
            canSave = (
                this.state.username !== this.props.username ||
                this.state.password !== this.props.password ||
                this.state.is_admin !== this.props.is_admin ||
                this.state.token !== this.props.token
            );
        }

        if(canSave && (this.state.username !== this.props.username) && this.props.userKnown(this.state.username)){
            canSave = false;
        }

        canSave = canSave ? this.fieldsPopulated() : false

        return(
                            <Form>
                                    <InputGroup>
                                        <Form.Control
                                            placeholder={this.state.username || "Username"}
                                            value={this.state.username}
                                            onChange={(e) => {this.fieldChange("username", e)}}
                                            disabled={this.state.mode !== "edit" && this.state.mode !== "new"}
                                        />
                                        { this.state.mode !== "new" && <Button
                                            variant={"secondary"}
                                            size={"sm"}
                                            disabled={this.state.username === ""}
                                            onClick={() => {
                                                navigator.clipboard.writeText(this.props.username);
                                                this.props.sendAlert({
                                                    variant:"success",
                                                    heading:"Username copied to clipboard",
                                                    message:this.props.username,
                                                    timeout:2});
                                            }}
                                        >
                                            <ClipboardFill size={18} data-toggle={"tooltip"} title={"Copy Username"}/>
                                        </Button> }
                                    </InputGroup>
                                    <InputGroup className={"mt-2 mb-1"}>
                                        <Form.Control
                                            placeholder={this.state.password || "Password"}
                                            value={this.state.password}
                                            type={this.state.show_password ? "" : "password"}
                                            onChange={(e) => {this.fieldChange("password", e)}}
                                            disabled={this.state.mode !== "edit" && this.state.mode !== "new"}
                                        />
                                        {this.state.mode !== "new" && <Button
                                            variant={"secondary"}
                                            size={"sm"}
                                            disabled={this.state.password === ""}
                                            onClick={() => {
                                                navigator.clipboard.writeText(this.props.password);
                                                this.props.sendAlert({
                                                    variant:"success",
                                                    heading:"Password copied to clipboard",
                                                    message:this.props.username,
                                                    timeout:2});
                                            }}
                                        >
                                            <ClipboardFill size={18} data-toggle={"tooltip"} title={"Copy Password"}/>
                                        </Button>}
                                    </InputGroup>
                                    <InputGroup className={"mt-2 mb-1"}>
                                        <Form.Control
                                            placeholder={this.state.token || "Token"}
                                            value={this.state.token}
                                            type={this.state.show_password ? "" : "password"}
                                            onChange={(e) => {this.fieldChange("token", e)}}
                                            disabled={this.state.mode !== "edit" && this.state.mode !== "new"}
                                        />
                                        {this.state.mode !== "new" &&
                                        <Button
                                            variant={"secondary"}
                                            size={"sm"}
                                            disabled={this.state.token === ""}
                                            onClick={() => {
                                                navigator.clipboard.writeText(this.props.token);
                                                this.props.sendAlert({
                                                    variant:"success",
                                                    heading:"Token copied to clipboard",
                                                    message:this.props.username,
                                                    timeout:2});
                                            }}
                                        >
                                            <ClipboardFill size={18} data-toggle={"tooltip"} title={"Copy Token"}/>
                                        </Button>}
                                    </InputGroup>
                                    <Form.Group className={"mb-1"}>
                                        <Form.Check
                                            type={"checkbox"}
                                            label={"Is Admin"}
                                            checked={this.state.is_admin}
                                            onChange={(event) => {
                                                this.setState({is_admin: !this.state.is_admin})
                                            }}
                                            disabled={this.state.mode !== "edit" && this.state.mode !== "new"}
                                        />
                                    </Form.Group>
                                <UserButtonGroup
                                    mode={this.state.mode}
                                    changeMode={this.changeMode}
                                    username={this.state.username}
                                    password={this.state.password}
                                    token={this.state.token}
                                    sendAlert={this.props.sendAlert}
                                    deleteUser={this.delete}
                                    isNew={this.props.isNew}
                                    showPassword={this.toggleShowPassword}
                                    passwordShown={this.state.show_password}
                                    canSave={canSave}
                                />
                            </Form>
        );
    }
}

export class UserCard extends UserForm {
    render(){
        return <Card body className={"m-3"}>
            { super.render() }
        </Card>
    };
}
