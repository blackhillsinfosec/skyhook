import './App.css';
import { Auth } from "./modules/auth";
import { AlertPanel } from "./modules/alert_panel";
import React from "react";
import Container from 'react-bootstrap/Container';
import {fileApi} from "./modules/file_api";
import {BrowserPane} from "./modules/browser_pane";

class App extends React.Component {

    constructor(props){
        super(props);
        this.state={

            // Used to determine if we're working in an authenticated
            // state. If not, the authentication form should be displayed
            // until credentials are provided.
            authenticated: false,

            // Alert is the current alert being displayed.
            alert: null,

            // Mode indicates the visible display. One of:
              // - home
              // - users
              // - obfuscators
            mode: "users",

        };

        this.sendAlert = this.sendAlert.bind(this);
        this.setMode = this.setMode.bind(this);
        this.componentDidMount = this.componentDidMount.bind(this);
        this.sendObfsConfig = this.sendObfsConfig.bind(this);
    }

    componentDidMount(){

        //================================================================
        // SET AN INTERVAL TO PERPETUALLY CHECK FOR AN AUTHENTICATED STATE
        //================================================================

        if(!this.auth_interval) {
            console.log("Setting auth interval");

            this.auth_interval = setInterval(() => {

                if(fileApi.token && !this.state.authenticated) {

                    console.log("Setting authenticated")
                    this.setState({authenticated: true})

                } else if(!fileApi.token && this.state.authenticated) {

                    console.log("Setting unauthenticated")
                    this.setState({authenticated: false})
                    localStorage.clear()

                };
            }, 1000)
        }
    }

    // setMode is used to determine which interface panel is
    // to be displayed.
    setMode(mode){
        this.setState({mode:mode})
    }

    // Send an alert.
    sendAlert(alert){
        this.setState({
            alert:alert,

        });
    }

    sendObfsConfig(){
        let api_config = JSON.parse(localStorage.getItem("api_config") || '{"obfuscators":[]}');
        return api_config.obfuscators;
    }

    recvObfsConfig(obfs_config){
        let api_config = JSON.parse(localStorage.getItem("api_config"));
        api_config.obfuscators = obfs_config;
        localStorage.setItem("api_config", JSON.stringify(api_config));
    }

    render() {

        //=======================
        // MANAGE THE ALERT PANEL
        //=======================

        let alert;
        if(this.state.alert){
            alert = (
                <AlertPanel
                    className={"m-4 fixed-bottom"}
                    variant={this.state.alert.variant}
                    heading={this.state.alert.heading}
                    message={this.state.alert.message}
                    show={this.state.alert.show}
                    timeout={this.state.alert.timeout}
                    timeoutCallback={() => {
                        this.setState({alert: null});
                    }}
                />
            )
        }

        return (
            <React.Fragment>
                <Auth sendAlert={this.sendAlert} show={!this.state.authenticated}/>
                <Container className={this.state.authenticated ? "" : "d-none"}>
                    {alert && alert}
                    {this.state.authenticated && <BrowserPane
                        recvObfsConfig={this.sendObfsConfig}
                        sendObfsConfig={this.recvObfsConfig}
                        authenticated={this.state.authenticated}
                        sendAlert={this.sendAlert}
                    />}
                </Container>
            </React.Fragment>
        );
    }
}

export default App;
