import React from "react";
import {Button, Modal, Form, ButtonGroup} from "react-bootstrap";
import {fileApi} from "./file_api";
import { Alert } from "react-bootstrap";

export class Auth extends React.Component {

    constructor(props){
        super(props);
        this.state={
            authenticated: false,
            username: "",
            password: "",
            token: "",
            token_expiration: "",
            show: true,
            alert: null,
        }
        this.authenticate = this.authenticate.bind(this);
        this.updateField = this.updateField.bind(this);
    }

    async authenticate(){

        localStorage.setItem("user_token", this.state.token);
        let o = await fileApi.postLogin({
            username: this.state.username,
            password: this.state.password,
            token: this.state.token,
        })

        if(o.ok){
            // TODO manage the token
            this.setState({
                show: false,
                authenticated: true,
                alert: o.output.alert,
                username: "",
                password: "",
                token: ""});
        } else {
            // Reset the form
            this.setState({
                username: "",
                password: "",
                token: "",
                alert: o.output.alert})
        }
    }

    updateField(name, event){
        this.setState({[name]:event.target.value})
    }

    render(){

        let alert;
        if(this.state.alert && !this.state.authenticated){
            let heading;

            if(this.state.alert.header){
                heading = <Alert.Heading>{this.state.alert.header}</Alert.Heading>;
            }

            alert = (
                <Alert
                    variant={this.state.alert.variant}
                    onClose={() => {this.setState({alert: null})}}
                    className={"m-2"}
                    dismissible
                >
                    {heading}
                    <p>{this.state.alert.message}</p>
                </Alert>
            )

            setTimeout(() => {
                this.setState({alert: null})
            }, 5*1000)
        }

        return (
          <Modal
              show={!fileApi.token}
              onHide={this.authenticate}
              backdrop={"static"}
              keyboard={false}>
              { alert }
              <Modal.Body>
                  <Form.Group className={"mb-3"} controlId={"login-username"}>
                      <Form.Control
                          placeholder={"Username"}
                          value={this.state.username}
                          onChange={(e) => {this.updateField("username", e)}}
                      />
                  </Form.Group>
                  <Form.Group className={"mb-3"} controlId={"login-password"}>
                      <Form.Control
                          placeholder={"Password"}
                          type={"password"}
                          value={this.state.password}
                          onChange={(e) => {this.updateField("password", e)}}
                      />
                  </Form.Group>
                  <Form.Group className={"mb-3"} controlId={"login-token"}>
                      <Form.Control
                          placeholder={"Token"}
                          type={"password"}
                          value={this.state.token}
                          onChange={(e) => {this.updateField("token", e)}}
                      />
                  </Form.Group>
              </Modal.Body>
              <Modal.Footer>
                  <ButtonGroup>
                      <Button
                          variant={"primary"}
                          onClick={this.authenticate}
                          disabled={!(this.state.username && this.state.password)}
                      >Submit</Button>
                  </ButtonGroup>
              </Modal.Footer>
          </Modal>
        );
    }
}
