FROM node:16 as builder
WORKDIR /workdir
COPY ./ /workdir/
RUN yarn install --pure-lockfile && yarn build

FROM node:16-stretch-slim as deploy
WORKDIR /app
COPY --from=builder /workdir/build ./build
COPY --from=builder /workdir/env.js ./

RUN npm install -g serve

ARG PORT=5000
ENV PORT ${PORT}

EXPOSE ${PORT}

CMD ["sh", "-c", "node ./env.js && exec serve -s build -l ${PORT}"]
