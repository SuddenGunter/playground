#include <stdio.h>
#include <math.h>                       // <-- added for powf
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_log.h"
#include "i2c_bus.h"
#include "bmp280.h"

// I²C configuration – adjust to your board
#define I2C_MASTER_NUM           I2C_NUM_0
#define I2C_MASTER_SDA_IO        8
#define I2C_MASTER_SCL_IO        9
#define I2C_MASTER_FREQ_HZ       100000

static const char *TAG = "bmp280_example";

void app_main(void)
{
        ESP_LOGI(TAG, "Starting app");

    // 1. Configure and create the I²C bus
    i2c_config_t i2c_conf = {
        .mode = I2C_MODE_MASTER,
        .sda_io_num = I2C_MASTER_SDA_IO,
        .scl_io_num = I2C_MASTER_SCL_IO,
        .sda_pullup_en = GPIO_PULLUP_ENABLE,
        .scl_pullup_en = GPIO_PULLUP_ENABLE,
        .master.clk_speed = I2C_MASTER_FREQ_HZ,
    };

    i2c_bus_handle_t bus = i2c_bus_create(I2C_MASTER_NUM, &i2c_conf);
    if (bus == NULL)
    {
        ESP_LOGE(TAG, "Failed to create I²C bus");
        return;
    }

    // 2. Create BMP280 sensor handle (use default address 0x76)
    bmp280_handle_t bmp = bmp280_create(bus, 0x77);
    if (bmp == NULL)
    {
        ESP_LOGE(TAG, "Failed to create BMP280 device");
        i2c_bus_delete(&bus);
        return;
    }

    // 3. Initialise the sensor with default settings
    esp_err_t err = bmp280_default_init(bmp);
    if (err != ESP_OK)
    {
        ESP_LOGE(TAG, "BMP280 initialisation failed: %s", esp_err_to_name(err));
        bmp280_delete(&bmp);
        i2c_bus_delete(&bus);
        return;
    }

    ESP_LOGI(TAG, "BMP280 initialised successfully");

    // 4. Read sensor data periodically
    while (1)
    {
        float temperature = 0.0f;
        float pressure = 0.0f;
        float humidity = 0.0f;
        float altitude = 0.0f;

        // Read temperature
        if (bmp280_read_temperature(bmp, &temperature) != ESP_OK)
        {
            ESP_LOGE(TAG, "Failed to read temperature");
        }

        // Read pressure
        if (bmp280_read_pressure(bmp, &pressure) != ESP_OK)
        {
            ESP_LOGE(TAG, "Failed to read pressure");
        }

        // Read humidity
        if (bmp280_read_humidity(bmp, &humidity) != ESP_OK)
        {
            ESP_LOGE(TAG, "Failed to read humidity");
        }

        // Calculate altitude (sea‑level pressure = 1013.25 hPa)
        // NOTE: The original bmp280_read_altitude() has a bug – it reads pressure into the wrong
        // variable. We therefore calculate altitude manually here.
        float sea_level_hpa = 1013.25f;
        if (pressure > 0.0f)
        {
            altitude = 44330.0f * (1.0f - powf(pressure / sea_level_hpa, 0.1903f));
        }

        ESP_LOGI(TAG,
                 "Temp: %.2f °C, Pressure: %.2f hPa, Humidity: %.2f %%, Altitude: %.2f m",
                 temperature, pressure, humidity, altitude);

        vTaskDelay(pdMS_TO_TICKS(2000));
    }
}