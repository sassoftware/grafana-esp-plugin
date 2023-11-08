FROM grafana/grafana:9.5.13-ubuntu

USER root
WORKDIR /root

RUN apt-get -y update
RUN apt-get -y install git build-essential

RUN curl -L https://go.dev/dl/go1.21.3.linux-amd64.tar.gz > go1.21.3.linux-amd64.tar.gz

RUN rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.21.3.linux-amd64.tar.gz

RUN touch README; printf "~~~~~~ START THE DLV SERVER WITH THIS COMMAND BEFORE RUNNING IDE DEBUGGER ~~~~~~ \r\ndlv attach --headless --api-version 2 --accept-multiclient --listen=:3222 \$(pgrep -f sasesp-plugin)\r\n\r\n" >> README

RUN echo "export PATH=$PATH:/usr/local/go/bin:~/go/bin" >> ~/.bashrc
RUN echo "cat ~/README" >> ~/.bashrc

RUN /usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest
RUN git clone https://github.com/magefile/mage; \
    cd mage; \
    export PATH=$PATH:/usr/local/go/bin; \
    go run bootstrap.go

CMD cd /var/lib/grafana/plugins/sasesp-plugin; \
    mage build:debug; \
	mage reloadPlugin; \
	dlv attach --headless --api-version 2 --accept-multiclient --listen=:3222 $(pgrep -f sasesp-plugin)
