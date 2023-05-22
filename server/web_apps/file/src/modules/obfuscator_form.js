
import React from "react";
import {Row,Col,Form,Button,ButtonGroup, Alert, FloatingLabel} from "react-bootstrap";
import {QuestionCircleFill} from "react-bootstrap-icons";

export class ObfuscatorForm extends React.Component {

    constructor(props){
        super(props);
        this.state = {
            disabled: true,
            config: this.props.config ? JSON.stringify(this.props.config) : JSON.stringify([]),
            show_help: false
        }
        this.toggleDisabled = this.toggleDisabled.bind(this);
    }

    toggleDisabled(){
        // TODO implement checks to ensure there aren't any
        //  ongoing file transfers!
        //  We should disallow changes until all are cleared.
        if(!this.state.disabled){
            if(this.props.canConfigure){
                try {
                    let config = JSON.parse(this.state.config);
                    this.props.sendDisabledUpdate(false);
                    this.props.sendFormUpdate(config);
                    this.setState({disabled: true, config: JSON.stringify(config)});
                    this.props.sendAlert({
                        variant: "success",
                        message: "Obfuscation configuration updated.",
                        timeout: 5
                    })
                } catch(e) {
                    this.props.sendAlert({
                        variant: "danger",
                        heading: "Failure",
                        message: "Failed to parse new obfuscation config into JSON.",
                        timeout: 10,
                    });
                }
            } else {
                this.props.sendAlert({
                    variant: "danger",
                    heading: "Failure",
                    message: "Can't apply changes while file transfers are ongoing.",
                    timeout: 10,
                });
            }
        }
        this.setState({disabled: !this.state.disabled})
    }

    render(){

        let no_edit_warning;
        if(!this.props.canConfigure){
            no_edit_warning = (
                <Row>
                    <Col>
                    <Alert variant={"warning"}>
                        Obfuscator can't be configured while file transfers are ongoing.
                    </Alert>
                    </Col>
                </Row>);
        }

        let cancel_button;
        if(!this.state.disabled){
            cancel_button = (
                <Button
                    size={"sm"}
                    onClick={() => {
                        this.setState({
                            config: this.props.config ? JSON.stringify(this.props.config) : JSON.stringify([{}]),
                            disabled: true
                        })
                    }}
                >
                    Cancel
                </Button>
            )
        }

        let help;
        if(this.state.show_help){
            help = (
                <p>
                    This configuration is used to determine the sequence
                    of obfuscation algorithms that will be applied to key
                    request elements.
                    <br/><br/>
                    The server-side configuration is applied via the <b>
                    admin server</b>, which also provides a copy button.
                    <br/><br/>
                    Use the copy button in the admin server to get
                    a value that can be copied into the form below.
                </p>
            );
        }

        return(
            <div>
                <Row>
                    <h2>Obfuscation Configuration <QuestionCircleFill
                        size={15}
                        onClick={(e) => {
                            this.setState({show_help: !this.state.show_help})
                        }}/></h2>
                    {help}
                </Row>
                {no_edit_warning}
                <Row>
                    <Col>
                        <Form>
                            <FloatingLabel label={"Obfuscator Configuration"}>
                                <Form.Control
                                    as={"textarea"}
                                    placeholder={"Paste configuration here"}
                                    style={{ height: '200px'}}
                                    disabled={this.state.disabled}
                                    onChange={(e) => {
                                        this.setState({
                                            config: e.target.value,
                                        })
                                    }}
                                    value={this.state.config}
                                />
                            </FloatingLabel>
                        </Form>
                    </Col>
                </Row>
                <Row className={"justify-content-end"}>
                    <Col className={
                        this.state.disabled ? "col-md-1" : "col-md-2"
                    }>
                        <ButtonGroup className={"mt-2"}>
                            <Button
                                disabled={!this.props.canConfigure}
                                onClick={this.toggleDisabled}
                                size={"sm"}
                                variant={
                                    this.state.disabled ? "primary" : "warning"
                                }
                            >
                                {this.state.disabled ? "Edit" : "Save"}
                            </Button>
                            {cancel_button}
                        </ButtonGroup>
                    </Col>
                </Row>
            </div>
        );
    }
}
