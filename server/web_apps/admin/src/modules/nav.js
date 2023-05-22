import React from "react";
import Container from 'react-bootstrap/Container';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import {Dropdown} from "react-bootstrap";
import {adminApi} from "./admin_api";
import {CodeSlash, Link, Link45deg} from "react-bootstrap-icons";

export class ShNavbar extends React.Component {
    constructor(props){
        super(props);
        this.shutdown = this.shutdown.bind(this);
        this.logout = this.logout.bind(this);
    }

    async shutdown(){
        let out = await adminApi.postShutdown();
        if(out.output.success){
            this.props.sendAlert(
                {
                    variant:"danger",
                    message:"Server is shutting down.",
                    timeout:5,
                    show:true,
                }
            );
        } else {
            this.props.sendAlert(out.alert);
        }
    }

    async logout(){
        let out = await adminApi.postLogout();
        localStorage.clear();
        this.props.sendAlert(out.alert);
    }

    render() {

        let fqdn_links = [];
        let i=-1;
        for (const [fqdn, sets] of Object.entries(this.props.links)){

            i+=1
            let variant="primary"
            switch(i){
                case(1):
                    variant="success";
                    break;
                case(2):
                    variant="danger";
                    break;
                case(3):
                    variant="warning";
                    break;
                case(4):
                    variant="info";
                    break;
                default:
                    i=0;
                    variant="primary";
            }
            variant = "link-"+variant;

            fqdn_links.push(
                <Dropdown.Item className={variant} disabled={true} key={`${fqdn}-header`}>{fqdn}</Dropdown.Item>
            );
            fqdn_links.push(
                <Dropdown.Item
                    onClick={(e) => {
                        navigator.clipboard.writeText(sets.standard.html);
                        this.props.sendAlert({
                            variant: "primary",
                            message: "Link copied to clipboard.",
                            timeout: 3
                        })
                    }} key={`${fqdn}-std-landing-page`}>
                    <Link size={20} className={variant}/> Standard Landing Page
                </Dropdown.Item>
            )
            fqdn_links.push(
                <Dropdown.Item
                    onClick={(e) => {
                        navigator.clipboard.writeText(sets.encrypted.autoload_html);
                        this.props.sendAlert({
                            variant: "primary",
                            message: "Link copied to clipboard.",
                            timeout: 3
                        })
                    }} key={`${fqdn}-enc-landing-page`}>
                    <Link45deg size={20} className={variant}/> Encrypted Landing Page
                </Dropdown.Item>
            )
            fqdn_links.push(
                <Dropdown.Item
                    onClick={(e) => {
                        navigator.clipboard.writeText(sets.encrypted.html);
                        this.props.sendAlert({
                            variant: "primary",
                            message: "Link copied to clipboard.",
                            timeout: 3
                        })
                    }} key={`${fqdn}-enc-blank-landing-page`}>
                    <Link45deg size={20} className={variant}/> Blank Landing Page (for JS Loader)
                </Dropdown.Item>
            )
            fqdn_links.push(
                <Dropdown.Item
                    onClick={(e) => {
                        adminApi.getEncryptedJs().then((o) => {
                            navigator.clipboard.writeText(o.output.encrypted_js);
                            this.props.sendAlert({
                                variant: "primary",
                                message: "JS content copied to clipboard.",
                                timeout: 3
                            })
                        })
                    }} key={`${fqdn}-enc-js-loader`}>
                    <CodeSlash size={17} className={variant}/> Encrypted JS Loader
                </Dropdown.Item>
            )
        }

        return (
            <Navbar bg={"light"} expand={"lg"}>
                <Container>
                    <Navbar.Brand href={"#home"}>Skyhook Admin Panel</Navbar.Brand>
                    <Navbar.Toggle aria-controls={"basic-navbar-nav"}/>
                    <Navbar.Collapse id={"navbar-nav"}>
                        <Nav className={"me-auto"}>
                            <Nav.Link
                                href={"#users"}
                                onClick={(e) => {
                                    e.preventDefault();
                                    this.props.sendMode("users")
                                }}
                            >
                                Users
                            </Nav.Link>
                            <Nav.Link
                                href={"#obfuscation"}
                                onClick={(e) => {
                                    e.preventDefault();
                                    this.props.sendMode("obfuscators");
                                }}
                            >
                                Obfuscators
                            </Nav.Link>

                            <Dropdown as={Nav.Item}>
                                <Dropdown.Toggle as={Nav.Link}>Quick Copy</Dropdown.Toggle>
                                <Dropdown.Menu>
                                    <Dropdown.Item
                                        onClick={(e) => {
                                            e.preventDefault();
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
                                            })}}
                                    >
                                        Obfuscators
                                    </Dropdown.Item>
                                    {fqdn_links}
                                </Dropdown.Menu>
                            </Dropdown>

                        </Nav>
                        <Nav>
                            <Nav.Link
                                href={"#logout"}
                                onClick={(e) => {
                                    e.preventDefault();
                                    this.logout(e);
                                }}>
                                Logout
                            </Nav.Link>
                        </Nav>
                    </Navbar.Collapse>
                </Container>
            </Navbar>);
    }
}