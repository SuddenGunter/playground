#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"
#include "esp_system.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_log.h"
#include "nvs_flash.h"
#include "esp_netif.h"
#include "esp_http_client.h"
#include "esp_crt_bundle.h"
#include "esp_netif_sntp.h"   // for esp_netif_sntp_init, esp_netif_sntp_deinit
#include "esp_sntp.h"         // for sntp_get_sync_status, sntp_set_timezone, etc.

#define WIFI_SSID      CONFIG_ESP_WIFI_SSID
#define WIFI_PASS      CONFIG_ESP_WIFI_PASSWORD

static EventGroupHandle_t s_wifi_event_group;
#define WIFI_CONNECTED_BIT BIT0

static const char *TAG = "HTTPS_EXAMPLE";

// Needed for TLS
static void sync_time(void) {
    sntp_setoperatingmode(SNTP_OPMODE_POLL);
    sntp_setservername(0, "pool.ntp.org");
    sntp_init();

    // Wait up to 5 seconds for sync
    int retry = 0;
    while (sntp_get_sync_status() == SNTP_SYNC_STATUS_RESET && retry < 10) {
        vTaskDelay(pdMS_TO_TICKS(500));
        retry++;
    }

    if (retry == 10) {
        ESP_LOGW(TAG, "NTP sync failed – HTTPS may still fail");
    } else {
        time_t now;
        time(&now);
        ESP_LOGI(TAG, "Time synchronized: %s", ctime(&now));
    }
}

// HTTP Event Handler: streams data directly to serial output
esp_err_t _http_event_handler(esp_http_client_event_t *evt) {
if (evt->event_id == HTTP_EVENT_ON_DATA) {
        printf("%.*s", evt->data_len, (char *)evt->data);
    }
    return ESP_OK;}

// HTTPS GET Task executed only after Wi-Fi connects
static void https_get_task(void *pvParameters)
{
    esp_http_client_config_t config = {
        .url = "https://ifconfig.co/json",
        .crt_bundle_attach = esp_crt_bundle_attach,
        .event_handler = _http_event_handler,
        .timeout_ms = 10000,
    };

    esp_http_client_handle_t client = esp_http_client_init(&config);

    if (client == NULL) {
        ESP_LOGE(TAG, "Failed to initialize HTTP client");
        vTaskDelete(NULL);
        return;
    }

    esp_http_client_set_header(
        client,
        "User-Agent",
        "ESP32-HTTP-Client"
    );

    ESP_LOGI(TAG, "Fetching IP info from ifconfig.co/json...");

    esp_err_t err = esp_http_client_perform(client);

    if (err == ESP_OK) {
        ESP_LOGI(TAG,
                 "HTTPS Status = %d, content_length = %" PRId64,
                 esp_http_client_get_status_code(client),
                 esp_http_client_get_content_length(client));
    } else {
        ESP_LOGE(TAG,
                 "HTTPS request failed: %s",
                 esp_err_to_name(err));
    }

    esp_http_client_cleanup(client);
    vTaskDelete(NULL);
}

// Wi-Fi and IP Event Handler
static void wifi_event_handler(void* arg, esp_event_base_t event_base,
                               int32_t event_id, void* event_data) {
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        ESP_LOGI(TAG, "Wi-Fi disconnected. Reconnecting...");
        esp_wifi_connect();
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t* event = (ip_event_got_ip_t*) event_data;
        ESP_LOGI(TAG, "Wi-Fi Connected! IP: " IPSTR, IP2STR(&event->ip_info.ip));
        xEventGroupSetBits(s_wifi_event_group, WIFI_CONNECTED_BIT);
    }
}

void app_main(void) {
    // 1. Initialize NVS
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    // 2. Initialize Wi-Fi and TCP/IP stack
    s_wifi_event_group = xEventGroupCreate();
    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());
    esp_netif_create_default_wifi_sta();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID, &wifi_event_handler, NULL, NULL));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP, &wifi_event_handler, NULL, NULL));

    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASS,
        },
    };
    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_STA));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_STA, &wifi_config));
    ESP_ERROR_CHECK(esp_wifi_start());

    // 3. Block main until Wi-Fi connects and acquires an IP
    EventBits_t bits = xEventGroupWaitBits(
        s_wifi_event_group,
        WIFI_CONNECTED_BIT,
        pdFALSE,
        pdFALSE,
        portMAX_DELAY
    );

    // 4. Trigger HTTPS GET task only after connection success
    if (bits & WIFI_CONNECTED_BIT) {
        sync_time();
        xTaskCreate(&https_get_task, "https_get_task", 8192, NULL, 5, NULL);
    }
}