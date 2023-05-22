import React from "react";
import {
    Row,
    Col,
    Card,
    Form,
    Button,
    ButtonGroup,
    Popover,
    OverlayTrigger
} from "react-bootstrap"
import {AlertPanel} from "./alert_panel";
import {
    ArrowDownShort,
    ArrowUpShort,
    CheckLg,
    PencilFill,
    XLg
} from "react-bootstrap-icons";

export class Obfuscator extends React.Component {

    constructor(props){
        super(props);
        this.state = {
            alert: null,
            config: this.props.config,
            mode:"normal",
        };

        this.updateField = this.updateField.bind(this);
        this.revertConfig = this.revertConfig.bind(this);
        this.save = this.save.bind(this);
    }

    updateField(name, value){
        let buff = Object.assign({}, this.state.config);
        buff[name] = value;
        this.setState({config:buff});
    }

    revertConfig(){
        this.setState({config:this.props.config});
    }

    save(){
        this.props.sendUpdate(this.props.ind, this.state.config);
        this.setState({mode:"normal"});
    }

    render(){

        if(Object.keys(this.props.getObfs()).length === 0){
            return <div/>;
        }

        //================================
        // OBFUSCATOR CONFIGURATION FIELDS
        //================================

        let conf_desc = this.props.getObfs()[this.props.algo];

        let fields=[];
        let keys=Object.keys(this.state.config);
        for(let i=0; i<keys.length; i++){
            let name=keys[i];

            let field = this.state.config[name];
            let field_type = conf_desc[name] !== undefined ? typeof(field) : "text";

            if(field_type === "string"){
                field_type="text";
            }


            fields.push(
                <fieldset disabled={this.state.mode !== "edit"} key={`${this.props.algo}-${name}-field`}>
                <Form.Group className={keys.length > 1 && i !== keys.length-1 ? "mb-2" : ""}>
                    <Form.Label>{name.slice(0,1).toUpperCase()+name.slice(1,)}</Form.Label>
                    <Form.Control
                        placeholder={name}
                        type={field_type}
                        value={field}
                        onChange={(e)=>{
                            this.updateField(
                                name,
                                e.target.value)}}
                    />
                </Form.Group>
                </fieldset>
            );
        }

        //=================
        // OBFUSCATOR ALERT
        //=================

        let alert;
        if(this.state.alert){
            alert=(
                <AlertPanel
                    variant={this.state.alert.variant}
                    message={this.state.alert.message}
                    header={this.state.alert.header}
                    timeout={this.state.alert.timeout}
                    show={true}
                />
            )
        }

        //====================
        // SAVE/CANCEL BUTTONS
        //====================

        let save;
        let cancel;
        if(this.state.mode === "edit"){
            save = (
                <Button
                    variant={"warning"}
                    onClick={() => {
                        this.save();
                        this.setState({mode:"normal"})}
                }>
                    <CheckLg size={19}/>
                </Button>);
            cancel = (
                <Button
                    variant={"success"}
                    onClick={() => {
                        this.revertConfig(this.props.ind, this.props.config);
                        this.setState({mode:"normal"});
                    }}>
                    <XLg size={15}/>
                </Button>
            )
        }

        //====================
        // EDIT/DELETE BUTTONS
        //====================

        let edit;
        let deleteB;
        if(this.state.mode === "normal" && !this.props.reordering){
            edit = (
                <Button
                    onClick={() => {this.setState({mode: "edit"})}}
                    variant={"warning"}
                    className={"text-nowrap"}>
                    <PencilFill size={15}/>
                </Button>
            );

            let dPop = (
                <Popover>
                    <Popover.Header as={"h3"}>Delete obfuscator?</Popover.Header>
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
                                    onClick={() => {this.props.sendDel(this.props.ind)}}
                                >
                                    Confirm
                                </Button>
                            </Col>
                        </Row>
                    </Popover.Body>
                </Popover>
            );

            deleteB = (
                <OverlayTrigger
                    trigger={"focus"}
                    placement={"bottom"}
                    overlay={dPop}
                >
                    <Button
                        className={"text-dark text-nowrap"}
                        variant={"danger"}
                        text={"dark"}
                    >
                        <XLg size={15}/>
                    </Button>
                </OverlayTrigger>
            );
        }

        //=====================
        // MOVE UP/DOWN BUTTONS
        //=====================

        let move_up;
        let move_down;
        if(this.props.reordering){
            move_up = (<Button
                variant={"secondary"}
                disabled={this.props.first}
                onClick={() => {
                    this.props.move(this.props.ind, "up");
                }}>
                <ArrowUpShort size={17}/>
            </Button>);
            move_down = (<Button
                variant={"secondary"}
                onClick={() => {
                    this.props.move(this.props.ind, "down");
                }}
                disabled={this.props.last}>
                <ArrowDownShort size={17}/>
            </Button>);
        }

        return (
            <Row className={"mb-3"}>
                <Col>
                <Card>
                    <Card.Header>
                        <Row>
                            <Col className={"col pt-1 text-nowrap"}>
                        <b>Stage {this.props.ind+1}:</b> {this.props.algo.slice(0,1).toUpperCase() + this.props.algo.slice(1)}
                            </Col>
                            <Col className={"d-flex col justify-content-end"}>
                                <ButtonGroup
                                    size={"sm"}
                                >
                                    {save && save}
                                    {cancel && cancel}
                                    {edit && edit}
                                    {deleteB && deleteB}
                                    {move_up && move_up}
                                    {move_down && move_down}
                                </ButtonGroup>
                            </Col>
                        </Row>
                    </Card.Header>
                    <Card.Body>
                        {alert && alert}
                        <Form children={fields}/>
                    </Card.Body>
                </Card>
                </Col>
            </Row>
        );
    }
}