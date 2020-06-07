This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app).

## Local development

### Running 

Add .env.local and the other variants as per this [guide](https://facebook.github.io/create-react-app/docs/adding-custom-environment-variables)

The following environment variables are used and shown below with examples:
```
REACT_APP_SERVER_URL=http://localhost:8080
REACT_APP_UPDATE_FREQUENCY_MILLISECONDS=1000
```

## Deployment

A dockerfile is provided.

## TODO

lack of feedback when interacting with buttons (if you click deploy, the frontend will silently update and change the button text from Redeploy tag_value to Deploy tag_value)

tag values in a card's dropdown menu should reflect which one is deployed with a highlight ideally

show nomad deployment status as a color on each card

autocomplete options do not push the cards out of the way but covers them instead.

add repo name next to tag (multiple repos may be associated with 1 tag) when showing autocomplete options
