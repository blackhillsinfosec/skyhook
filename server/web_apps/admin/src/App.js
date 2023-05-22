import './App.css';
import { Auth } from "./modules/auth";
import { ShNavbar } from './modules/nav.js'
import { UserPanel } from "./modules/user/user_panel";
import { AlertPanel } from "./modules/alert_panel";
import {ObfsPanel} from "./modules/obfs_panel";
import React from "react";
import Container from 'react-bootstrap/Container';
import {adminApi} from "./modules/admin_api";

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

            // Obfuscators contains a list of configured obfuscators.
            obfuscators: [],

            // Mode indicates the visible display. One of:
              // - home
              // - users
              // - obfuscators
            mode: "users",
            links: {},
        };

        this.sendAlert = this.sendAlert.bind(this);
        this.setMode = this.setMode.bind(this);
        this.componentDidMount = this.componentDidMount.bind(this);
    }

    componentDidMount(){

        //================================================================
        // SET AN INTERVAL TO PERPETUALLY CHECK FOR AN AUTHENTICATED STATE
        //================================================================

        if(!this.auth_interval) {
            console.log("Setting auth interval");

            this.auth_interval = setInterval(() => {

                if(adminApi.token && !this.state.authenticated) {

                    console.log("Setting authenticated");
                    this.setState({authenticated: true});
                    this.getLinks();

                } else if(!adminApi.token && this.state.authenticated) {

                    console.log("Setting unauthenticated");
                    this.setState({authenticated: false});

                };
            }, 1000)
        }
    }

    getLinks(){
        adminApi.getAdminLinks().then((o) => {
            if(!o.output.success){
                this.sendAlert(o.alert);
            }else{
                this.setState({links:o.output.links})
            }
        })
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
                {alert && alert}
                <Auth show={!this.state.authenticated}/>
                <div className={this.state.authenticated ? "" : "d-none"}>
                <ShNavbar sendAlert={this.sendAlert} sendMode={this.setMode} links={this.state.links}/>
                <Container>
                    <UserPanel
                        show={this.state.mode === "users"}
                        sendAlert={this.sendAlert}
                        users={this.state.users}
                        isNew={false}
                        isAuthenticated={this.state.authenticated}
                        sendMode={this.setMode}/>
                    <ObfsPanel
                        show={this.state.mode === "obfuscators"}
                        isAuthenticated={this.state.authenticated}
                        sendAlert={this.sendAlert}
                    />
                </Container>
                </div>
            </React.Fragment>
        );
    }
}

export default App;
