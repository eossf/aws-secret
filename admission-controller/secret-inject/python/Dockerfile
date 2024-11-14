FROM ubuntu:xenial-20210114

ADD . /

RUN apt-get update -y && \
    apt-get install -y python-pip python-dev

RUN pip install --upgrade pip && pip install -r requirements.txt

ENTRYPOINT [ "python" ]

CMD [ "admission_controller.py" ]