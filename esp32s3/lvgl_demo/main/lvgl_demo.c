#include <stdio.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

#include "driver/gpio.h"
#include "driver/spi_master.h"

#include "esp_lcd_panel_io.h"
#include "esp_lcd_panel_ops.h"
#include "esp_lcd_ili9341.h"

#include "esp_lvgl_port.h"
#include "lvgl.h"

#define PIN_NUM_SCLK      12
#define PIN_NUM_MOSI      11
#define PIN_NUM_MISO      13
#define PIN_NUM_LCD_DC     9
#define PIN_NUM_LCD_RST   14
#define PIN_NUM_LCD_CS    10

#define LCD_HOST          SPI2_HOST
#define LCD_H_RES         240
#define LCD_V_RES         320

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
        spi_bus_initialize(LCD_HOST, &buscfg, SPI_DMA_CH_AUTO)
    );

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
            io_handle
        )
    );

    // ILI9341
    esp_lcd_panel_dev_config_t panel_config = {
        .reset_gpio_num = PIN_NUM_LCD_RST,
        .rgb_ele_order = LCD_RGB_ELEMENT_ORDER_RGB,
        .bits_per_pixel = 16,
    };

    ESP_ERROR_CHECK(
        esp_lcd_new_panel_ili9341(
            *io_handle,
            &panel_config,
            panel_handle
        )
    );

    ESP_ERROR_CHECK(esp_lcd_panel_reset(*panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_init(*panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_disp_on_off(*panel_handle, true));
}

void app_main(void)
{
    esp_lcd_panel_io_handle_t io_handle = NULL;
    esp_lcd_panel_handle_t panel_handle = NULL;

    init_display(&io_handle, &panel_handle);

    // Start LVGL port
    const lvgl_port_cfg_t lvgl_cfg = ESP_LVGL_PORT_INIT_CONFIG();

    ESP_ERROR_CHECK(lvgl_port_init(&lvgl_cfg));

    // Attach the esp_lcd display to LVGL
    const lvgl_port_display_cfg_t display_cfg = {
        .io_handle = io_handle,
        .panel_handle = panel_handle,

        // Start with a partial framebuffer.
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
            .swap_bytes = false,
            #endif
        },

        #if LVGL_VERSION_MAJOR >= 9
        .color_format = LV_COLOR_FORMAT_RGB565_SWAPPED,
        #endif
    };

    lv_display_t *display = lvgl_port_add_disp(&display_cfg);

    assert(display != NULL);

    // Now use normal LVGL APIs.
    if (lvgl_port_lock(0)) {

       static lv_style_t style_grad;

    static bool inited = false;

    if(!inited) {
        lv_style_init(&style_grad);
        lv_style_set_radius(&style_grad, 20);
        lv_style_set_bg_opa(&style_grad, (255 * 100 / 100));
        lv_style_set_bg_color(&style_grad, lv_color_hex(0x6366f1));
        lv_style_set_bg_grad_color(&style_grad, lv_color_hex(0xec4899));
        lv_style_set_bg_grad_dir(&style_grad, LV_GRAD_DIR_VER);
        lv_style_set_bg_main_stop(&style_grad, 80);
        lv_style_set_bg_grad_stop(&style_grad, 220);
        lv_style_set_shadow_color(&style_grad, lv_color_hex(0x6366f1));
        lv_style_set_shadow_width(&style_grad, 26);
        lv_style_set_shadow_offset_y(&style_grad, 10);
        lv_style_set_shadow_opa(&style_grad, 70);
        lv_style_set_text_color(&style_grad, lv_color_hex(0xffffff));

        inited = true;
    }

    lv_obj_t * screen = lv_screen_active();

    /* 💡 Slide `bg_main_stop`/`bg_grad_stop` (0..255) to move where the indigo→pink blend starts and ends. */
    lv_obj_t * container = lv_obj_create(screen);
    lv_obj_set_size(container, 210, 150);
    lv_obj_set_align(container, LV_ALIGN_CENTER);
    lv_obj_add_style(container, &style_grad, 0);
    lv_obj_t * label = lv_label_create(container);
    lv_obj_set_align(label, LV_ALIGN_CENTER);
    lv_label_set_text(label, "Gradient");
        lvgl_port_unlock();
    }
}