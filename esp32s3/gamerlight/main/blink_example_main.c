/* Blink Example – Breathing Rainbow (Brightness ≤ 0.5) */
#include <stdio.h>
#include <math.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "led_strip.h"
#include "sdkconfig.h"

static const char *TAG = "example";
#define BLINK_GPIO CONFIG_BLINK_GPIO

static float s_hue = 0.0f; // Current hue angle (0–360°)

/* HSV → RGB conversion (hexagonal cone algorithm) */
static void hsv_to_rgb(float h, float s, float v, uint8_t *r, uint8_t *g, uint8_t *b)
{
    // v (brightness) will NEVER exceed 0.5 in our main loop
    if (s == 0.0f)
    {
        *r = *g = *b = (uint8_t)(v * 255);
        return;
    }
    h = fmodf(h, 360.0f);
    float c = v * s;
    float x = c * (1.0f - fabsf(fmodf(h / 60.0f, 2.0f) - 1.0f));
    float m = v - c;
    float r1, g1, b1;
    if (h < 60)
    {
        r1 = c;
        g1 = x;
        b1 = 0;
    }
    else if (h < 120)
    {
        r1 = x;
        g1 = c;
        b1 = 0;
    }
    else if (h < 180)
    {
        r1 = 0;
        g1 = c;
        b1 = x;
    }
    else if (h < 240)
    {
        r1 = 0;
        g1 = x;
        b1 = c;
    }
    else if (h < 300)
    {
        r1 = x;
        g1 = 0;
        b1 = c;
    }
    else
    {
        r1 = c;
        g1 = 0;
        b1 = x;
    }
    *r = (uint8_t)((r1 + m) * 255);
    *g = (uint8_t)((g1 + m) * 255);
    *b = (uint8_t)((b1 + m) * 255);
}

#ifdef CONFIG_BLINK_LED_STRIP
static led_strip_handle_t led_strip;

static void blink_led(uint8_t r, uint8_t g, uint8_t b)
{
    // GRB fix for most on‑board WS2812/SK6812 (swap Red & Green)
    led_strip_set_pixel(led_strip, 0, g, r, b);
    led_strip_refresh(led_strip);
}

static void configure_led(void)
{
    ESP_LOGI(TAG, "Configuring addressable LED");
    led_strip_config_t strip_config = {
        .strip_gpio_num = BLINK_GPIO,
        .max_leds = 1,
    };
#if CONFIG_BLINK_LED_STRIP_BACKEND_RMT
    led_strip_rmt_config_t rmt_config = {
        .resolution_hz = 10 * 1000 * 1000,
        .flags.with_dma = false,
    };
    ESP_ERROR_CHECK(led_strip_new_rmt_device(&strip_config, &rmt_config, &led_strip));
#elif CONFIG_BLINK_LED_STRIP_BACKEND_SPI
    led_strip_spi_config_t spi_config = {
        .spi_bus = SPI2_HOST,
        .flags.with_dma = true,
    };
    ESP_ERROR_CHECK(led_strip_new_spi_device(&strip_config, &spi_config, &led_strip));
#else
#error "unsupported LED strip backend"
#endif
    led_strip_clear(led_strip);
}
#endif

void app_main(void)
{
    configure_led();

    // ---------- Breathing & colour speed controls ----------
    float hue_step = 0.2f;      // 0.2° per step → ~30 sec for full rainbow
    float breath_speed = 0.03f; // Radians per step (smaller = slower pulse)
    float breath_angle = 0.0f;  // Tracks sine wave position
    int delay_ms = 20;          // Update every 20 ms

    while (1)
    {
        // ---- 1. Compute brightness using a sine wave ----
        // sinf() goes from -1.0 to +1.0
        // (sin + 1) / 2  →  goes from 0.0 to 1.0
        float raw_brightness = (sinf(breath_angle) + 1.0f) / 2.0f;

        // ---- 2. Clamp brightness with custom min and max ----
        float min_brightness = 0.02f; // Never fully off (2% glow)
        float max_brightness = 0.25f; // Peak at 25% brightness (less bright than 50%)

        float brightness = min_brightness + (raw_brightness * (max_brightness - min_brightness));
        // Now ranges from 0.02 → 0.25

        // ---- 3. Convert current Hue to RGB using the clamped brightness ----
        uint8_t r, g, b;
        hsv_to_rgb(s_hue, 1.0f, brightness, &r, &g, &b);

        // ---- 4. Send to the LED ----
        blink_led(r, g, b);

        // ---- 5. Advance the hue (color) ----
        s_hue += hue_step;
        if (s_hue >= 360.0f)
            s_hue -= 360.0f;

        // ---- 6. Advance the breathing angle ----
        breath_angle += breath_speed;
        if (breath_angle > (2.0f * M_PI))
            breath_angle -= (2.0f * M_PI);

        // ---- 7. Wait before next update ----
        vTaskDelay(pdMS_TO_TICKS(delay_ms));
    }
}