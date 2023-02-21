FROM alpine

RUN apk add --no-cache curl
RUN apk add --no-cache jq
RUN apk add --no-cache bash

COPY realm-export.json realm-export.json
COPY import.sh import.sh
RUN chmod 777 ./import.sh

CMD ["bash", "./import.sh"]
