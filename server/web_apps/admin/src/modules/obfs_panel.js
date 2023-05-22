import React from "react";
import {Row, Col, ButtonGroup, Button, DropdownButton, Dropdown} from "react-bootstrap";
import {Obfuscator} from "./obfuscator";
import {Check, ClipboardFill, Intersect, Lock} from "react-bootstrap-icons";
import {adminApi} from "./admin_api";

export class ObfsPanel extends React.Component {

    constructor(props){
        super(props);
        this.state = {
            obfs: [],
            obfs_avail: {},
            mode: "normal",
        }

        this.init_load=false;
        this.clear_int=null;

        this.mvObfs = this.mvObfs.bind(this);
        this.recvUpdate = this.recvUpdate.bind(this);
        this.delObf = this.delObf.bind(this);
        this.getObfsConfig = this.getObfsConfig.bind(this);
        this.getObfsAvail = this.getObfsAvail.bind(this);
        this.shareAvailObfs = this.shareAvailObfs.bind(this);
        this.addObf = this.addObf.bind(this);
        this.putObfs = this.putObfs.bind(this);
    }

    componentDidMount(){
        this.clear_int = setInterval(() => {
            if(this.state.obfs && !this.props.isAuthenticated){
                this.setState({obfs:[]});
                this.init_load=false;
            } else if(!this.init_load && this.props.isAuthenticated){
                this.init_load=true;
                this.getObfsConfig();
                this.getObfsAvail();
            }
        }, 1000)
    }

    componentWillUnmount() {
        if(this.clear_int){
            clearInterval(this.clear_int);
            this.clear_int=null;
        }
    }

    async putObfs(){
        let out = await adminApi.putObfsConfig({obfuscators:this.state.obfs});
        if(out.output.success){
            this.props.sendAlert({
                    variant: "success",
                    message: "Obfuscators saved",
                    timeout: 5,
                    show: true
                });
        } else {
            this.props.sendAlert(out.output.alert);
        }
    }

    shareAvailObfs(){
        return this.state.obfs_avail;
    }

    addObf(name){
        let obfs = Object.assign([], this.state.obfs);
        obfs.push({algo: name, config: this.state.obfs_avail[name]});
        this.setState({obfs:obfs});
    }

    async getObfsConfig(){
        let out = await adminApi.getObfsConfig();
        if(out.output.success){
            this.setState({obfs:out.output.obfuscators ? out.output.obfuscators : []});
        } else {
            this.props.sendAlert(out.output.alert);
        }
    }

    async getObfsAvail(){
        let out = await adminApi.getObfsAvail();
        if(out.output.success){
            this.setState({obfs_avail:out.output.obfuscators})
        } else {
            this.props.sendAlert(out.output.alert);
        }
    }

    delObf(ind){
        let buff = Object.assign([], this.state.obfs);
        buff.splice(ind, 1);
        this.setState({
            obfs:buff,
        });
    }

    recvUpdate(ind, config){
        let buff = Object.assign([], this.state.obfs);
        buff[ind].config = config;
        this.setState({obfs:buff});
    }

    mvObfs(ind, dir){

        dir = dir === "up" ? -1 : 1

        let tail = Object.assign([], this.state.obfs);
        let head = tail.splice(0, ind+1);
        let target = head.pop()

        // lol so bad
        if(dir === -1){
            let buff = head.pop();
            head.push(target);
            head.push(buff);
            head = head.concat(tail)
        } else {
            head.push(tail.splice(0, 1)[0]);
            head.push(target);
            head = head.concat(tail);
        }

        this.setState({obfs: head});

    }

    render(){

        //============
        // OBFUSCATORS
        //============

        let obfuscators=[];
        for(let i=0; i<this.state.obfs.length; i++){
            let obf=this.state.obfs[i];
            obfuscators.push(
                <Obfuscator
                    key={`${obf.algo}-${i}-obfs-${String(Date.now())}`}
                    ind={i}
                    algo={obf.algo}
                    config={obf.config}
                    first={i === 0}
                    last={i === this.state.obfs.length-1}
                    move={this.mvObfs}
                    reordering={this.state.mode === "reorder"}
                    sendUpdate={this.recvUpdate}
                    sendDel={this.delObf}
                    getObfs={this.shareAvailObfs}
                />
            );
        }

        //===============
        // REORDER BUTTON
        //===============

        let reorder;
        if(this.state.mode === "normal" && this.state.obfs.length > 1){
            reorder = (
                <Button
                    variant={"secondary"}
                    onClick={() => {
                        this.setState({mode:"reorder"})
                    }}
                >
                    <Intersect size={20}/> Reorder
                </Button>
            );
        }else if (this.state.mode === "reorder") {
            reorder = (
                <Button
                    variant={"secondary"}
                    onClick={() => {
                        this.setState({mode:"normal"});
                    }}>
                    <Lock size={20}/> Keep Reorder
                </Button>
            );
        }

        //=================
        // ADD/SAVE BUTTONS
        //=================

        let add;
        let save;
        let copy;
        if(this.state.mode !== "reorder"){
            let links=[];
            let keys=Object.keys(this.state.obfs_avail);
            for(let i=0; i<keys.length; i++){
                let key=keys[i];
                links.push(
                   <Dropdown.Item
                       eventKey={`${key}-add-algo`}
                       key={`${key}-add-algo`}
                       onClick={() => {this.addObf(key)}}
                   >
                       {key.slice(0,1).toUpperCase()+key.slice(1)}
                   </Dropdown.Item>
                );
            }
            add = (
                <DropdownButton
                    size={"sm"}
                    variant={"secondary"}
                    as={ButtonGroup}
                    title={"Add"}
                >
                    {links}
                </DropdownButton>
            );
            save = (
                <Button
                    size={"sm"}
                    variant={"secondary"}
                    disabled={this.state.mode === "reorder"}
                    onClick={this.putObfs}
                >
                    <Check size={20}/> Save
                </Button>
            );
            copy = (
                <Button
                  size={"sm"}
                  variant={"secondary"}
                  onClick={() => {
                      adminApi.getObfsConfig().then((e) => {
                          if(e.output.success){
                              navigator.clipboard.writeText(JSON.stringify(e.output.obfuscators));
                              this.props.sendAlert({
                                  variant: "primary",
                                  message: "Obfuscator config copied",
                                  timeout: 3})
                          }else{
                              this.props.sendAlert(e.output.alert);
                          }
                      })
                  }}>
                    <ClipboardFill size={15} data-toggle={"tooltip"} title={"Copy Configuration"}/> Copy Config
                </Button>
            );

        }

        return(
            <div className={this.props.show ? "" : "d-none" }>
                <Row className={"row mt-3"}>
                    <Col className={"col-1"}>
                        <h1>Obfuscators</h1>
                    </Col>
                    <Col className={"d-flex justify-content-end pt-1 me-2 mb-3"}>
                        <ButtonGroup size={"sm"}>
                            { add && add }
                            { reorder }
                            { copy && copy }
                            { save && save }
                        </ButtonGroup>
                    </Col>
                </Row>
                <Row className={"row pt-2 justify-content-center"}>
                    <Col className={"col-md-6"}>
                        {obfuscators}
                    </Col>
                </Row>
            </div>
        );
    }

}
