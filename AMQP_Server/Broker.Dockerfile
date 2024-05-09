FROM rabbitmq:3.10.7-management-alpine as base

WORKDIR /rabbitmq

RUN addgroup qlay

RUN chown -R rabbitmq:qlay /rabbitmq
RUN chmod -R gu+rwx /rabbitmq
RUN chown -R rabbitmq:qlay /var/lib/rabbitmq

RUN mkdir -p /data/log
RUN mkdir -p /data/mnesia

RUN chown -R rabbitmq:qlay /data/mnesia
RUN chmod -R gu+rwx /data/mnesia

RUN chown -R rabbitmq:qlay /data/log
RUN chmod -R gu+rwx /data/log

USER rabbitmq

# Copy rabbitmq.conf
COPY --chown=rabbitmq:qlay rabbitmq.conf /etc/rabbitmq/rabbitmq.conf

RUN rabbitmq-plugins enable --offline rabbitmq_mqtt rabbitmq_management && rabbitmq-plugins enable rabbitmq_mqtt