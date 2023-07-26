FROM node:6.14.2
EXPOSE 80

ARG script
RUN echo $script > server.js

CMD [ "node", "server.js" ]