import React from "react";
import {Row, Col, Form, Button} from "react-bootstrap";

export class MiscConfigs extends React.Component{

    constructor(props){
        super(props);
        this.state={};
    }

    render(){
        return(
            <div>
                <Row>
                    <h2>Misc</h2>
                </Row>
                <Row>
                    <Col>
                        <Form>
                            <Form.Check
                                type={"switch"}
                                id={"advanced-switch"}
                                label={"Enable Staging"}
                                checked={this.props.stagingEnabled}
                                onChange={(e) => {
                                    this.props.toggleStaging();
                                }}
                            />
                        </Form>
                        <Button
                            size={"sm"}
                            className={"mt-2 mb-2"}
                            variant={"danger"}
                            onClick={(e) => {
                                e.preventDefault();
                                this.props.doLogout();
                            }}>
                            Logout
                        </Button>
                    </Col>
                </Row>
            </div>
        );
    }
}