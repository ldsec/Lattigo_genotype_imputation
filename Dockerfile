FROM golang:1.12

RUN apt-get update
RUN apt-get -y install python3 python3-pip
RUN pip3 install numpy
RUN pip3 install pandas
RUN pip3 install scikit-learn
RUN pip3 install matplotlib

COPY . /root/idash19_Track2

WORKDIR /root/idash19_Track2

ENTRYPOINT ["/bin/bash"]
