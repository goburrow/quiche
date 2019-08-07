#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <unistd.h>
#include "quiche.h"

quiche_config *default_config() {
    quiche_config *config = quiche_config_new(QUICHE_PROTOCOL_VERSION);
    if (config == NULL) {
        fprintf(stderr, "failed to create config\n");
        exit(-1);
    }
    int ret = quiche_config_set_application_protos(config,
        (uint8_t *) "\x06proto1\x06proto2", 14);
    if (ret != 0) {
        fprintf(stderr, "set application protos: %d\n", ret);
        exit(-1);
    }
    ret = quiche_config_load_cert_chain_from_pem_file(config,
        "quiche/examples/cert.crt");
    if (ret != 0) {
        fprintf(stderr, "load cert: %d\n", ret);
        exit(-1);
    }
    ret = quiche_config_load_priv_key_from_pem_file(config,
        "quiche/examples/cert.key");
    if (ret != 0) {
        fprintf(stderr, "load key: %d\n", ret);
        exit(-1);
    }
    quiche_config_set_initial_max_data(config, 30);
    quiche_config_set_initial_max_stream_data_bidi_local(config, 15);
    quiche_config_set_initial_max_stream_data_bidi_remote(config, 15);
    quiche_config_set_initial_max_stream_data_uni(config, 10);
    quiche_config_set_initial_max_streams_bidi(config, 3);
    quiche_config_set_initial_max_streams_uni(config, 3);
    quiche_config_verify_peer(config, false);
    return config;
}

ssize_t recv_send(quiche_conn* conn, uint8_t *buf, ssize_t buf_size, ssize_t len) {
    ssize_t left = len;
    while (left > 0) {
        ssize_t read = quiche_conn_recv(conn, &buf[len-left], left);
        if (read == QUICHE_ERR_DONE) {
            break;
        }
        if (read < 0) {
            return read;
        }
        left -= read;
    }
    ssize_t off = 0;
    while (off < buf_size) {
        ssize_t write = quiche_conn_send(conn, &buf[off], buf_size-off);
        if (write == QUICHE_ERR_DONE) {
            break;
        }
        if (write < 0) {
            return write;
        }
        off += write;
    }
    return off;
}

void debug_log(const char *line, void *argp) {
    fprintf(stderr, "%s\n", line);
}

int main() {
    static uint8_t buf[65535];
    uint8_t client_cid[4] = {1};
    uint8_t server_cid[4] = {2};

    quiche_enable_debug_logging(debug_log, NULL);
    quiche_config *config = default_config();

    client_cid[0] = 1;
    quiche_conn *client = quiche_connect("", (const uint8_t *) client_cid, sizeof(client_cid), config);
    if (client == NULL) {
        fprintf(stderr, "failed to create connection\n");
        return -1;
    }
    server_cid[0] = 2;
    quiche_conn *server = quiche_accept((const uint8_t *) server_cid, sizeof(server_cid), (const uint8_t *) NULL, 0, config);
    if (server == NULL) {
        fprintf(stderr, "failed to create connection\n");
        return -1;
    }
    ssize_t len = quiche_conn_send(client, buf, sizeof(buf));
    if (len < 0) {
        fprintf(stderr, "send: %zd\n", len);
        return -1;
    }
    fprintf(stdout, "client sent %zd bytes\n", len);
    while (!quiche_conn_is_established(client) && !quiche_conn_is_established(server)) {
        len = recv_send(server, buf, sizeof(buf), len);
        if (len < 0) {
            fprintf(stderr, "server recv/send: %zd\n", len);
            return -1;
        }
        fprintf(stdout, "server sent %zd bytes\n", len);
        len = recv_send(client, buf, sizeof(buf), len);
        if (len < 0) {
            fprintf(stderr, "client recv/send: %zd\n", len);
            return -1;
        }
        fprintf(stdout, "client sent %zd bytes\n", len);
    }
    len = recv_send(server, buf, sizeof(buf), len);
    if (len < 0) {
        fprintf(stderr, "server recv/send: %zd\n", len);
        return -1;
    }
    fprintf(stdout, "connected\n");
    return 0;
}
