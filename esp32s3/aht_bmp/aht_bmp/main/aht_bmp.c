#include <stdio.h>
#include <math.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

#include "esp_log.h"
#include "esp_err.h"

#include "i2c_bus.h"
#include "aht20.h"
#include "bmp280.h"

// ============================================================
// I2C configuration
// ============================================================

#define I2C_MASTER_NUM       I2C_NUM_0
#define I2C_MASTER_SDA_IO CONFIG_I2C_MASTER_SDA
#define I2C_MASTER_SCL_IO CONFIG_I2C_MASTER_SCL
#define I2C_MASTER_FREQ_HZ  100000

// BMP280 I2C address
#define BMP280_I2C_ADDR     0x77

static const char *TAG = "SENSOR_APP";


void app_main(void)
{
    ESP_LOGI(TAG, "Starting AHT20 + BMP280 application");

    // ========================================================
    // 1. Create shared I2C bus
    // ========================================================

    i2c_config_t i2c_conf = {
        .mode = I2C_MODE_MASTER,
        .sda_io_num = I2C_MASTER_SDA_IO,
        .scl_io_num = I2C_MASTER_SCL_IO,
        .sda_pullup_en = GPIO_PULLUP_ENABLE,
        .scl_pullup_en = GPIO_PULLUP_ENABLE,
        .master.clk_speed = I2C_MASTER_FREQ_HZ,
    };

    i2c_bus_handle_t bus =
        i2c_bus_create(I2C_MASTER_NUM, &i2c_conf);

    if (bus == NULL)
    {
        ESP_LOGE(TAG, "Failed to create I2C bus");
        return;
    }

    ESP_LOGI(TAG, "I2C bus created successfully");

    // ========================================================
    // 2. Initialize AHT20
    // ========================================================

    aht20_i2c_config_t aht20_conf = {
        .bus_inst = bus,
        .i2c_addr = AHT20_ADDRRES_0,
    };

    aht20_dev_handle_t aht20 = NULL;

    esp_err_t err = aht20_new_sensor(&aht20_conf, &aht20);

    if (err != ESP_OK)
    {
        ESP_LOGE(TAG,
                 "Failed to initialize AHT20: %s",
                 esp_err_to_name(err));

        i2c_bus_delete(&bus);
        return;
    }

    ESP_LOGI(TAG, "AHT20 initialized successfully");

    // ========================================================
    // 3. Initialize BMP280
    // ========================================================

    bmp280_handle_t bmp280 =
        bmp280_create(bus, BMP280_I2C_ADDR);

    if (bmp280 == NULL)
    {
        ESP_LOGE(TAG, "Failed to create BMP280");

        i2c_bus_delete(&bus);
        return;
    }

    err = bmp280_default_init(bmp280);

    if (err != ESP_OK)
    {
        ESP_LOGE(TAG,
                 "BMP280 initialization failed: %s",
                 esp_err_to_name(err));

        bmp280_delete(&bmp280);
        i2c_bus_delete(&bus);
        return;
    }

    ESP_LOGI(TAG, "BMP280 initialized successfully");

    // ========================================================
    // 4. Sensor reading loop
    // ========================================================

    while (1)
    {
        // ----------------------------------------------------
        // AHT20
        // ----------------------------------------------------

        uint32_t temperature_raw = 0;
        uint32_t humidity_raw = 0;

        float aht20_temperature = 0.0f;
        float aht20_humidity = 0.0f;

        err = aht20_read_temperature_humidity(
            aht20,
            &temperature_raw,
            &aht20_temperature,
            &humidity_raw,
            &aht20_humidity
        );

        if (err != ESP_OK)
        {
            ESP_LOGE(TAG,
                     "AHT20 read failed: %s",
                     esp_err_to_name(err));
        }

        // ----------------------------------------------------
        // BMP280
        // ----------------------------------------------------

        float bmp_temperature = 0.0f;
        float pressure = 0.0f;
        float altitude = 0.0f;

        err = bmp280_read_temperature(
            bmp280,
            &bmp_temperature
        );

        if (err != ESP_OK)
        {
            ESP_LOGE(TAG, "BMP280 temperature read failed");
        }

        err = bmp280_read_pressure(
            bmp280,
            &pressure
        );

        if (err != ESP_OK)
        {
            ESP_LOGE(TAG, "BMP280 pressure read failed");
        }

        // ----------------------------------------------------
        // Calculate altitude
        // ----------------------------------------------------

        const float sea_level_hpa = 1013.25f;

        if (pressure > 0.0f)
        {
            altitude =
                44330.0f *
                (1.0f -
                 powf(pressure / sea_level_hpa, 0.1903f));
        }

        // ====================================================
        // Print combined sensor data
        // ====================================================

        ESP_LOGI(TAG,
                 "AHT20  -> Temp: %.2f °C, Humidity: %.2f %%",
                 aht20_temperature,
                 aht20_humidity);

        ESP_LOGI(TAG,
                 "BMP280 -> Temp: %.2f °C, Pressure: %.2f hPa, Altitude: %.2f m",
                 bmp_temperature,
                 pressure,
                 altitude);

        ESP_LOGI(TAG, "----------------------------------------");

        // Read every 2 seconds
        vTaskDelay(pdMS_TO_TICKS(2000));
    }
}