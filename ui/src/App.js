import React, { Component } from 'react';
import axios from "axios";
import PropTypes from 'prop-types';
import './App.css';
import { cloneDeep } from 'lodash';

import Grid from '@material-ui/core/Grid';
import { emphasize } from '@material-ui/core/styles/colorManipulator';
import { withStyles } from '@material-ui/core/styles';

import SearchBar from './components/SearchBar';
import Repository from './components/Repository';

function onlyUnique(value, index, self) {
    return self.indexOf(value) === index;
}

const styles = theme => ({
    root: {
        flexGrow: 1,
        height: 250,
    },
    input: {
        display: 'flex',
        padding: 0,
        height: 'auto',
    },
    valueContainer: {
        display: 'flex',
        flexWrap: 'wrap',
        flex: 1,
        alignItems: 'center',
        overflow: 'hidden',
    },
    chip: {
        margin: theme.spacing(0.5, 0.25),
    },
    chipFocused: {
        backgroundColor: emphasize(
            theme.palette.type === 'light' ? theme.palette.grey[300] : theme.palette.grey[700],
            0.08,
        ),
    },
    noOptionsMessage: {
        padding: theme.spacing(1, 2),
    },
    singleValue: {
        fontSize: 16,
    },
    placeholder: {
        position: 'absolute',
        left: 2,
        bottom: 6,
        fontSize: 16,
    },
    paper: {
        position: 'relative',
        zIndex: 1,
        marginTop: theme.spacing(1),
        left: 0,
        right: 0,
    },
    divider: {
        height: theme.spacing(2),
    },
});

class App extends Component {
    constructor(props) {
        super(props);

        this.state = {
            loading: true,
            originalRepos: {},
            repos: {},
            multi: [],
            query: "",
        };
    }

    componentDidMount() {
        this.getUpdate();
        this.timerID = setInterval(
            () => this.getUpdate(),
            window.config.env.REACT_APP_UPDATE_FREQUENCY_MILLISECONDS
        );
    }

    componentWillUnmount() {
        clearInterval(this.timerID);
    }

    async flipAutoDeploy(val, repoName) {
        const reqUrl = new URL(`/tags/${repoName}`, window.config.env.REACT_APP_SERVER_URL);
        axios.post(reqUrl, {"auto_deploy": val});
    }

    async resetToVersionedAutoDeployment(repoName) {
        const reqUrl = new URL(`/tags/${repoName}/reset`, window.config.env.REACT_APP_SERVER_URL);
        axios.post(reqUrl, {});
    }

    async postDeploy(pinnedTag, repoName) {
        const reqUrl = new URL(`/tags/${repoName}`, window.config.env.REACT_APP_SERVER_URL);
        axios.post(reqUrl, {"pinned_tag": pinnedTag});
    }

    getUpdate() {
        const reqUrl = new URL("/repos", window.config.env.REACT_APP_SERVER_URL);
        axios.get(reqUrl)
             .then((resp) => {
                 if (this.state.loading) {
                     this.setState({
                         loading: false,
                         repos: resp.data
                     });
                 }
                 var allTerms = [];
                 if (this.state.query !== "") {
                     allTerms = allTerms.concat(this.state.query.toLowerCase());
                 }
                 var multi = this.state.multi.map(option => option.label);
                 if (multi && multi.length) {
                     allTerms = allTerms.concat(multi);
                 }
                 this.setState({
                     repos: this.filterRepoMap(resp.data, allTerms),
                     originalRepos: resp.data
                 });
             });
    }

    flattenRepoMap = () => {
        var keys = Object.keys(this.state.originalRepos);
        var tags = Object.values(this.state.originalRepos).map(repo => repo.tags).flat().filter(onlyUnique);
        return keys.concat(tags);
    }

    // filters all elements in array `keys` by the following predicate:
    // if none of the strings in `filterValues` is a substring of that element
    filterArray = (keys, filterValues) => {
        return keys.filter((key) => {
            for (let val of filterValues) {
                if (key.search(val) !== -1) {
                    return true;
                }
            }
            return false;
        });
    }

    filterRepoMap = (hashMap, filterValues) => {
        if (filterValues.length === 0) {
            return hashMap;
        }
        var rtn = {};
        for (let key of Object.keys(hashMap)) {
            var values = this.filterArray(hashMap[key].tags, filterValues);
            if (values && values.length) {
                // one of the tags is in filterValues
                rtn[key] = hashMap[key];
                rtn[key].tags = values;
            } else if (this.filterArray([key], filterValues).length !== 0) {
                // none of the tags is in filterValues, but the key is
                rtn[key] = hashMap[key];
            }
        }
        return rtn;
    }

    componentWillMount() {
        this.setState({repos: this.state.originalRepos});
    }

    handleChangeMulti = (event) => {
        if (event == null) {
            event = [];
        }
        var multi = event.map(option => option.label);
        var original = cloneDeep(this.state.originalRepos);
        this.setState({
            multi: event,
            repos: this.filterRepoMap(original, multi)
        });
    }

    handleSearchInput = (event) => {
        if (event !== "") {
            var original = cloneDeep(this.state.originalRepos);
            var allTerms = [event.toLowerCase()];
            var multi = this.state.multi.map(option => option.label);
            if (multi && multi.length) {
                allTerms = allTerms.concat(multi);
            }
            this.setState({
                repos: this.filterRepoMap(original, allTerms),
                query: event.toLowerCase()
            });
        } else {
            if (Object.entries(this.state.repos).length === 0 || this.state.multi.length === 0) {
                this.setState({
                    repos: this.state.originalRepos,
                    query: ""
                });
            }
        }
    }

    render() {
        if (this.state.loading) {
            return <h2>Loading...</h2>;
        }

        const { classes } = this.props;
        return (
            <div className="App">
            <SearchBar
            classes={classes}
            onChange={this.handleChangeMulti}
            onInputChange={this.handleSearchInput}
            multi={this.state.multi}
            options={this.flattenRepoMap().map(option => ({
                value: option,
                label: option,
            }))}
            />
            <Grid container>
            {
                Object.entries(this.state.repos).map(([repoName, repo]) => {
                    return <Grid item xs={12} sm={6} lg={4} xl={3}>
                        <Repository key={repoName}
                                    name={repoName}
                                    repo={repo}
                                    postDeploy={this.postDeploy}
                                    resetToVersionedAutoDeployment={this.resetToVersionedAutoDeployment}
                                    flipAutoDeploy={this.flipAutoDeploy}></Repository>
                           </Grid>;
                })
            }
            </Grid>
            </div>
        );
    }
}

App.propTypes = {
    classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(App);
