/*
 * SPDX-FileCopyrightText: 2023-2025 Espressif Systems (Shanghai) CO LTD
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include <stdio.h>
#include "aht20.h"
#include "esp_system.h"
#include "esp_log.h"

#define I2C_MASTER_SCL_IO CONFIG_I2C_MASTER_SCL       /*!< gpio number for I2C master clock */
#define I2C_MASTER_SDA_IO CONFIG_I2C_MASTER_SDA        /*!< gpio number for I2C master data  */
#define I2C_MASTER_NUM I2C_NUM_0  /*!< I2C port number for master dev */
#define I2C_MASTER_FREQ_HZ 100000 /*!< I2C master clock frequency */
static const char *TAG = "aht20 test";
const i2c_config_t i2c_bus_conf = {
    .mode = I2C_MODE_MASTER,
    .sda_io_num = I2C_MASTER_SDA_IO,
    .sda_pullup_en = GPIO_PULLUP_ENABLE,
    .scl_io_num = I2C_MASTER_SCL_IO,
    .scl_pullup_en = GPIO_PULLUP_ENABLE,
    .master.clk_speed = I2C_MASTER_FREQ_HZ};

void app_main(void)
{
    i2c_bus_handle_t i2c_bus = i2c_bus_create(I2C_MASTER_NUM, &i2c_bus_conf);

    aht20_i2c_config_t i2c_conf = {
        .bus_inst = i2c_bus,         // I2C bus instance
        .i2c_addr = AHT20_ADDRRES_0, // Device address
    };
    aht20_dev_handle_t handle;

    // Block for 500ms.
    const TickType_t xDelay = 10000 / portTICK_PERIOD_MS;

    aht20_new_sensor(&i2c_conf, &handle);
    uint32_t temperature_raw, humidity_raw;
    float temperature, humidity;
    for (int i = 0; i < 10; i++) {
        vTaskDelay(xDelay);
        
    aht20_read_temperature_humidity(handle, &temperature_raw, &temperature, &humidity_raw, &humidity);
    ESP_LOGI(TAG, "%-20s: %2.2f %%", "humidity is", humidity);
    ESP_LOGI(TAG, "%-20s: %2.2f degC\n", "temperature is", temperature);
    }
}