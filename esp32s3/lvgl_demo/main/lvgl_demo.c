#include <stdio.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

#include "driver/gpio.h"
#include "driver/spi_master.h"
#include "driver/ledc.h"  // Added for PWM backlight control

#include "esp_lcd_panel_io.h"
#include "esp_lcd_panel_ops.h"
#include "esp_lcd_ili9341.h"

#include "esp_lvgl_port.h"
#include "lvgl.h"

#define PIN_NUM_SCLK 12
#define PIN_NUM_MOSI 11
#define PIN_NUM_MISO 13
#define PIN_NUM_LCD_DC 9
#define PIN_NUM_LCD_RST 14
#define PIN_NUM_LCD_CS 10
#define PIN_NUM_BK_LIGHT 4  // Backlight GPIO pin

#define LCD_HOST SPI2_HOST
#define LCD_H_RES 240
#define LCD_V_RES 320

// LEDC PWM settings
#define BK_LIGHT_LEDC_TIMER   LEDC_TIMER_0
#define BK_LIGHT_LEDC_MODE    LEDC_LOW_SPEED_MODE
#define BK_LIGHT_LEDC_CHANNEL LEDC_CHANNEL_0

static void init_backlight(void)
{
    // Configure PWM timer
    ledc_timer_config_t ledc_timer = {
        .speed_mode       = BK_LIGHT_LEDC_MODE,
        .timer_num        = BK_LIGHT_LEDC_TIMER,
        .duty_resolution  = LEDC_TIMER_10_BIT, // 0 - 1023 resolution
        .freq_hz          = 1000,              // 1 kHz flicker-free frequency
        .clk_cfg          = LEDC_AUTO_CLK
    };
    ESP_ERROR_CHECK(ledc_timer_config(&ledc_timer));

    // Configure PWM channel on GPIO 4
    ledc_channel_config_t ledc_channel = {
        .speed_mode     = BK_LIGHT_LEDC_MODE,
        .channel        = BK_LIGHT_LEDC_CHANNEL,
        .timer_sel      = BK_LIGHT_LEDC_TIMER,
        .intr_type      = LEDC_INTR_DISABLE,
        .gpio_num       = PIN_NUM_BK_LIGHT,
        .duty           = 0, // Initial duty cycle (off)
        .hpoint         = 0
    };
    ESP_ERROR_CHECK(ledc_channel_config(&ledc_channel));
}

// Helper to set backlight brightness from 0 to 100%
static void set_backlight_brightness(uint32_t percent)
{
    if (percent > 100) percent = 100;
    // Map 0-100% to 10-bit resolution (0-1023)
    uint32_t duty = (1023 * percent) / 100;
    ESP_ERROR_CHECK(ledc_set_duty(BK_LIGHT_LEDC_MODE, BK_LIGHT_LEDC_CHANNEL, duty));
    ESP_ERROR_CHECK(ledc_update_duty(BK_LIGHT_LEDC_MODE, BK_LIGHT_LEDC_CHANNEL));
}

static void init_display(esp_lcd_panel_io_handle_t *io_handle,
                         esp_lcd_panel_handle_t *panel_handle)
{
    // SPI bus
    spi_bus_config_t buscfg = {
        .sclk_io_num = PIN_NUM_SCLK,
        .mosi_io_num = PIN_NUM_MOSI,
        .miso_io_num = PIN_NUM_MISO,
        .data2_io_num = -1,
        .data3_io_num = -1,
        .max_transfer_sz = LCD_H_RES * 80 * sizeof(uint16_t),
    };

    ESP_ERROR_CHECK(
        spi_bus_initialize(LCD_HOST, &buscfg, SPI_DMA_CH_AUTO));

    // LCD I/O
    esp_lcd_panel_io_spi_config_t io_config = {
        .dc_gpio_num = PIN_NUM_LCD_DC,
        .cs_gpio_num = PIN_NUM_LCD_CS,
        .pclk_hz = 26 * 1000 * 1000,
        .lcd_cmd_bits = 8,
        .lcd_param_bits = 8,
        .spi_mode = 0,
        .trans_queue_depth = 10,
    };

    ESP_ERROR_CHECK(
        esp_lcd_new_panel_io_spi(
            (esp_lcd_spi_bus_handle_t)LCD_HOST,
            &io_config,
            io_handle));

    // ILI9341
    esp_lcd_panel_dev_config_t panel_config = {
        .reset_gpio_num = PIN_NUM_LCD_RST,
        .rgb_ele_order = LCD_RGB_ELEMENT_ORDER_BGR,
        .bits_per_pixel = 16,
    };

    ESP_ERROR_CHECK(
        esp_lcd_new_panel_ili9341(
            *io_handle,
            &panel_config,
            panel_handle));

    ESP_ERROR_CHECK(esp_lcd_panel_reset(*panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_init(*panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_disp_on_off(*panel_handle, true));
}

void app_main(void)
{
    esp_lcd_panel_io_handle_t io_handle = NULL;
    esp_lcd_panel_handle_t panel_handle = NULL;

    // 1. Initialize display and backlight
    init_display(&io_handle, &panel_handle);
    init_backlight();
    
    // Set initial brightness to 40% (adjust this value as needed)
    set_backlight_brightness(75);

    // 2. Start LVGL port
    const lvgl_port_cfg_t lvgl_cfg = ESP_LVGL_PORT_INIT_CONFIG();

    ESP_ERROR_CHECK(lvgl_port_init(&lvgl_cfg));

    // Attach the esp_lcd display to LVGL
    const lvgl_port_display_cfg_t display_cfg = {
        .io_handle = io_handle,
        .panel_handle = panel_handle,

        .buffer_size = LCD_H_RES * 40,
        .double_buffer = true,

        .hres = LCD_H_RES,
        .vres = LCD_V_RES,

        .rotation = {
            .swap_xy = false,
            .mirror_x = true,
            .mirror_y = false,
        },

        .flags = {
            .buff_dma = true,

#if LVGL_VERSION_MAJOR >= 9
            .swap_bytes = true,
#endif
        },

#if LVGL_VERSION_MAJOR >= 9
        .color_format = LV_COLOR_FORMAT_RGB565,
#endif
    };

    lv_display_t *display = lvgl_port_add_disp(&display_cfg);

    assert(display != NULL);

    // 3. Render UI
    if (lvgl_port_lock(0))
    {
        lv_obj_t *screen = lv_screen_active();

        /* Set screen background to Solarized Light base3 (#FDF6E3) */
        lv_obj_set_style_bg_color(screen, lv_color_hex(0xfaf0d4), 0);
        lv_obj_set_style_bg_opa(screen, LV_OPA_COVER, 0);

        /* Create "Hello World" label */
        lv_obj_t *label = lv_label_create(screen);
        lv_label_set_text(label, "Hello World");

        /* Position in top-left corner with 16px padding */
        lv_obj_align(label, LV_ALIGN_TOP_LEFT, 16, 16);

        /* Style text: Solarized Light base00 main text (#657B83) */
        lv_obj_set_style_text_color(label, lv_color_hex(0x657B83), 0);
        lv_obj_set_style_text_font(label, &lv_font_montserrat_28, 0);

        lvgl_port_unlock();
    }
}