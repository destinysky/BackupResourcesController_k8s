FROM python:3.9-slim
LABEL author="destinysky"
LABEL app="coldbackupWakeup"

#RUN pip3 install --no-cache-dir requests pymysql cryptography

COPY ./bin/coldbackup/coldbackup.py /home

WORKDIR /home
ENTRYPOINT ["python"]
CMD ["coldbackup.py"]