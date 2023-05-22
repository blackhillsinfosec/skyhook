import React from "react";
import {Alert} from "react-bootstrap";

export class AlertPanel extends React.Component {
    constructor(props){
        super(props);
        this.className = this.props.className ? this.props.className : "m-0";
        this.state = {
            show: this.props.show === undefined ? true : this.props.show
        };
        this.timeout_id = null;
        this.hideOrCallback = this.hideOrCallback.bind(this);
    }

    hideOrCallback(){
        if(this.timeout_id != null){
            clearTimeout(this.timeout_id);
        }
        if(this.props.timeoutCallback){
            this.props.timeoutCallback();
        } else {
            this.setState({show: false});
        }
    }

    render() {
        if(this.state.show){
            if(this.props.timeout) {
                this.timeout_id = setTimeout(this.hideOrCallback,this.props.timeout*1000);
            }
        }

        let heading, message;

        if(this.props.heading !== "" ){
            heading = <Alert.Heading>{this.props.heading}</Alert.Heading>
        }

        if(this.props.message !== ""){
            message = <p className={"m-0"}>{this.props.message}</p>
        }

        return (
            <Alert
                variant={this.props.variant}
                onClose={this.hideOrCallback}
                className={this.props.className}
                show={this.state.show}
                dismissible
            >
                {heading && heading}
                {message}
            </Alert>)
    }
}

