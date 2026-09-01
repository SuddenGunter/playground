idf_component_register(SRCS "station_example_main.c"
                       INCLUDE_DIRS "."
                       REQUIRES esp_http_client esp-tls mbedtls esp_wifi nvs_flash esp_event)