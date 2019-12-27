FROM golang:alpine

ARG modelPath
ENV MODEL_PATH=$modelPath

# CREATE APP DIRECTORY ("app" should be the name of your app's repository)
RUN mkdir -p /opt/tb
# Set CWD
WORKDIR /opt/tb
COPY . /opt/tb

RUN go get -v ./... && go build -o model $MODEL_PATH

CMD [ "./model" ]

