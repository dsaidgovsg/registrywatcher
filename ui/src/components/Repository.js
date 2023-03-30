import React, { Component } from 'react';

import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import Typography from '@material-ui/core/Typography';
import Select from '@material-ui/core/Select';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import FormControl from '@material-ui/core/FormControl';
import Button from '@material-ui/core/Button';
import CloudUploadIcon from '@material-ui/icons/CloudUpload';
import TextField from '@material-ui/core/TextField';
import Autocomplete from '@material-ui/lab/Autocomplete';

class Repository extends Component {
    constructor(props) {
        super(props);
        this.state = {
            selectedTag: this.props.repo.pinned_tag_value,
            inputTag: this.props.repo.pinned_tag_value
        };
    }

    handleChange = (event) => {
        this.setState({selectedTag: event.target.value});
    }

    render() {
        const showRedeployButton = this.state.selectedTag === this.props.repo.pinned_tag_value;
        let button;

        if (showRedeployButton) {
            button = <Button variant="contained"
                             color="default"
                             onClick={() => this.props.postDeploy(
                                 this.state.selectedTag, this.props.name)}>
                Redeploy {this.state.selectedTag}
                <CloudUploadIcon />
                     </Button>;
        } else {
            button = <Button variant="contained"
                             color="default"
                             onClick={() => this.props.postDeploy(
                                 this.state.selectedTag, this.props.name)}>
                Deploy {this.state.selectedTag}
                <CloudUploadIcon />
                     </Button>;
        }

        return (
            <Card className="foo">
                <div>
                    <CardContent>
                        <Typography component="h5" variant="h5">
                            {this.props.name}
                        </Typography>
                        <Typography variant="subtitle1" color="textSecondary">
                            {this.props.repo.pinned_tag === "" ? `Latest version: ${this.props.repo.pinned_tag_value}` : this.props.repo.pinned_tag}
                        </Typography>
                    </CardContent>
                </div>
                <div>
                    <FormControl>
                        <InputLabel htmlFor="age-simple">Tags</InputLabel>
                        <Autocomplete
                            value={this.state.selectedTag}
                            onChange={(event, newValue) => {
                                this.setState({selectedTag: newValue})
                            }}
                            inputValue={this.state.inputTag}
                            onInputChange={(event, newInputValue)=> {
                                if (newInputValue == '') {
                                    this.setState({
                                        inputTag: this.props.repo.pinned_tag_value,
                                        selectedTag: this.props.repo.pinned_tag_value
                                    })
                                } else {
                                    this.setState({
                                        inputTag: newInputValue,
                                    })
                                }
                                
                            }}
                            id="controlled-tags"
                            options={this.props.repo.tags}
                            style={{ width: 300 }}
                            renderInput={(params) => (
                                <TextField {...params} label="Tags"/>
                            )}
                        />
                    </FormControl>
                </div>
                <div>
                    <Button variant="contained"
                            color="default"
                            onClick={() => this.props.flipAutoDeploy(
                                !this.props.repo.auto_deploy, this.props.name)}>
                        AutoDeploy: {this.props.repo.auto_deploy.toString()}
                    </Button>
                    <Button variant="contained"
                            color="default"
                            onClick={() => this.props.resetToVersionedAutoDeployment(
                                this.props.name)}>
                        Reset (enable autodeploy, track versioned tags)
                    </Button>
                </div>
                <div>
                    {button}
                </div>
            </Card>
        );
    }
}

export default Repository;
