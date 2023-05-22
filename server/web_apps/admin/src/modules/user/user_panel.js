import React from "react";
import {UserCard, UserForm} from "./user";
import {Row, Col, Button, ButtonGroup, Popover, OverlayTrigger} from "react-bootstrap";
import {Plus} from "react-bootstrap-icons";
import {adminApi} from "../admin_api";

// UserPanel is the primary panel where all configured users are presented.
export class UserPanel extends React.Component {
    constructor(props){
        super(props);
        this.state = {
            users: [],
        };
        this.init_load = false;
        this.getUsers = this.getUsers.bind(this);
        this.getUsernames = this.getUsernames.bind(this);
        this.saveUsers = this.saveUsers.bind(this);
        this.userKnown = this.userKnown.bind(this);
        this.addUser = this.addUser.bind(this);
        this.deleteUser = this.deleteUser.bind(this);
        this.updateUser = this.updateUser.bind(this);
        this.clearUsers = this.clearUsers.bind(this);
    }

    async getUsers(){
        let out = await adminApi.getUsers();
        if(out.output.success){
            //this.props.sendAlert("success", "", "Updated user listing", 5);
            this.setState({users: out.output.users});
        } else {
            this.props.sendAlert(out.output.alert);
        }
    }

    // Retrieve all usernames from known users.
    getUsernames(){
        let vals=[];
        for(let i=0; i<this.state.users.length; i++){
            vals.push(this.state.users[i].username);
        }
        return vals;
    }

    async updateUser(oldUsername, user){
        let users = Object.assign([], this.state.users)
        let changed=false
        for(let i in users){
            let i_user = users[i]
            if (i_user.username === oldUsername){
                if(oldUsername !== user.username && this.userKnown(user.username)) {
                    this.props.sendAlert({variant: "warning", heading:"Username Error", message:"Username must be unique.", timeout: 5})
                    return false;
                } else {
                    changed=true
                    users[i] = user
                }
                break
            }
        }
        if(changed){
            let state = {users:users};
            let out = await adminApi.putUsers(state)
            if(out.output.success){
                this.props.sendAlert({
                    variant: "success",
                    heading: "User Updated",
                    message: user.username,
                    timeout: 5
                });
                this.setState(state);
            } else {
                changed=false;
                this.props.sendAlert(out.output.alert);
            }
        }
        return changed
    }

    async saveUsers(){
        // TODO finish the save method
        let out = await adminApi.putUsers({users: this.state.users})
        if(out.output.success){
            this.props.sendAlert({
                variant: "success",
                message: "Users saved",
                timeout: 5
            })
        } else {
            this.props.sendAlert(out.output.alert)
        }
    }

    // Add a user to the list of user values.
    async addUser(user){
        if(this.userKnown(user.username)){
            this.props.sendAlert({variant:"danger", heading:"Error",
                message: `Username is already known: ${user.username}`,
                timeout:5});
            return false;
        }

        //================
        // APPEND NEW USER
        //================

        let state = {};
        state.users = Object.assign([], this.state.users);
        state.users.push(user);

        //=======================
        // SEND UPDATED USER LIST
        //=======================

        let out = await adminApi.putUsers(state);
        if(out.output.success){
            this.props.sendAlert({
                variant: "success",
                heading: "User Created",
                message: user.username,
                timeout: 5
            });
        } else {
            this.props.sendAlert(out.output.alert);
        }

        this.setState(state);
        return true;
    }

    // Determine if a given user is known by username.
    userKnown(username){
        let known=this.getUsernames();
        if(known.indexOf(username)>-1){
            return true;
        }
        return false;
    }

    async deleteUser(username){
        let keep = [];
        for(let i=0; i<this.state.users.length; i++) {
            if (this.state.users[i].username !== username) {
                keep.push(this.state.users[i]);
            }
        }


        let out = await adminApi.putUsers({users: keep})
        if(out.output.success) {
            this.props.sendAlert({
                variant: "warning",
                heading: "User Deleted",
                message: username,
                timeout: 5});
            this.setState({
                users: keep,
            })
        } else {
            this.props.sendAlert(out.output.alert);
        }
    }

    clearUsers(){
        this.setState({users:[]});
    }

    componentDidMount() {
        this.clear_int = setInterval(() => {
            if(this.state.users && !this.props.isAuthenticated){
                this.setState({users:[]});
                this.init_load=false;
            } else if (!this.init_load && this.props.isAuthenticated){
                this.init_load=true;
                this.getUsers();
            }
        }, 1000);
    }

    componentWillUnmount() {
        if(this.clear_int){
            clearInterval(this.clear_int);
            this.clear_int = null;
        }
    }

    render(){

        if(!this.init_load && this.props.isAuthenticated){
            this.getUsers()
            this.init_load = true;
        }else if(!this.props.isAuthenticated && this.state.users){
            this.init_load = false;
        }

        //=====================
        // CREATE USER ELEMENTS
        //=====================

        let eles = []
        for(let i=0; i<this.state.users.length; i++){
            let u = this.state.users[i]
            eles.push(
            <UserCard
                key={`user-${u.username}`}
                user={u}
                sendAlert={this.props.sendAlert}
                deleteUser={this.deleteUser}
                username={u.username}
                password={u.password}
                is_admin={u.is_admin}
                token={u.token}
                updateUser={this.updateUser}
                userKnown={this.userKnown}
            />
            )
        }

        //===================
        // RENDER THE ELEMENT
        //===================

        return(
            <React.Fragment>
                <Row className={this.props.show ? "" : "d-none"}>
                    <Col className={"pt-2"}>
                        <h1>User Accounts</h1>
                    </Col>
                    <Col md={"auto"}>
                        <PanelControllerButtonGroup
                            addUser={this.addUser}
                            saveUsers={this.saveUsers}
                            getUsers={this.getUsers}
                            sendAlert={this.props.sendAlert}
                            userKnown={this.userKnown}
                        />
                    </Col>
                </Row>
                <Row
                    xs={1} md={3}
                    className={this.props.show ? "mt-1 g-4" : "d-none"}
                    children={eles}/>
            </React.Fragment>
        );
    }
}

class PanelControllerButtonGroup extends React.Component {
    render(){
        let pop = (
            <Popover>
                <Popover.Body>
                    <div className={"w-100"}>
                    <UserForm
                        key={`new-user-form`}
                        mode={"new"}
                        username={""}
                        password={""}
                        is_admin={false}
                        sendAlert={this.props.sendAlert}
                        addUser={this.props.addUser}
                        updateUser={this.updateUser}
                        userKnown={this.props.userKnown}
                    />
                    </div>
                </Popover.Body>
            </Popover>
        );

        return (
            <ButtonGroup
                aria-label={"user-panel-buttons"}
                className={"btn-group-sm mt-3"}
            >
                <OverlayTrigger
                    key={"new-user-overlay"}
                    trigger={"click"}
                    placement={"bottom"}
                    overlay={pop}>
                    <Button
                        variant={"secondary"}>
                        <Plus size={20}/> New User
                    </Button>
                </OverlayTrigger>
            </ButtonGroup>
        )
    }
}
